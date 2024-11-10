package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	baseURL      string
	webhookPort  string
	latestStatus string
	mu           sync.Mutex
	statusChan   chan string
}

// NewClient creates a new client to interact with the server
func NewClient(baseURL, webhookPort string) *Client {
	return &Client{
		baseURL:     baseURL,
		webhookPort: webhookPort,
		statusChan:  make(chan string, 1),
	}
}

// GetStatus checks the status of the job
func (c *Client) GetStatus() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/status", c.baseURL))
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var statusResponse struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(body, &statusResponse); err != nil {
		return "", err
	}

	c.mu.Lock()
	c.latestStatus = statusResponse.Status
	c.mu.Unlock()

	return statusResponse.Status, nil
}

// RegisterWebhook registers a webhook with the server
func (c *Client) RegisterWebhook(webhookURL string) error {
	data := map[string]string{"url": webhookURL}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(fmt.Sprintf("%s/register-webhook", c.baseURL), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to register webhook, status code: %d", resp.StatusCode)
	}

	return nil
}

// WebhookServer starts a local server to handle webhook notifications
func (c *Client) WebhookServer() {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		var webhookData map[string]string
		if err := json.NewDecoder(r.Body).Decode(&webhookData); err != nil {
			log.Println("Invalid webhook payload")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		c.mu.Lock()
		c.latestStatus = webhookData["status"]
		c.mu.Unlock()

		log.Printf("Webhook received: %v\n", webhookData)
		c.statusChan <- webhookData["status"]
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("Starting webhook server on port %s\n", c.webhookPort)
	log.Fatal(http.ListenAndServe(":"+c.webhookPort, nil))
}

// PollingStrategy encapsulates the polling logic for checking job status
func (c *Client) PollingStrategy(maxRetries int) (string, error) {
	baseInterval := 4 * time.Second

	for i := 0; i < maxRetries; i++ {
		status, err := c.GetStatus()
		if err != nil {
			log.Printf("Error getting status: %v\n", err)
			if i == maxRetries-1 {
				return "", err
			}
		} else {
			log.Printf("Job Status: %s\n", status)
			if status == "completed" || status == "error" {
				log.Println("Job completed via polling")
				return status, nil
			}
		}

		// Adaptive polling: reduce interval closer to the estimated completion
		interval := baseInterval
		if i > maxRetries/2 {
			interval = 2 * time.Second
		}
		if i > (3*maxRetries)/4 {
			interval = 1 * time.Second
		}

		log.Printf("Waiting for %v before next poll...\n", interval)
		time.Sleep(interval)
	}
	return "", fmt.Errorf("max retries reached without completion")
}

// StartClientWithWebhook starts the client library with webhook and polling
func (c *Client) StartClientWithWebhook() (string, error) {
	go c.WebhookServer()
	time.Sleep(2 * time.Second)

	webhookURL := fmt.Sprintf("http://localhost:%s/webhook", c.webhookPort)
	if err := c.RegisterWebhook(webhookURL); err != nil {
		return "", fmt.Errorf("error registering webhook: %v", err)
	}

	log.Println("Webhook registered, starting polling as backup...")

	// Start polling in a separate goroutine
	go func() {
		status, err := c.PollingStrategy(10)
		if err == nil && (status == "completed" || status == "error") {
			c.statusChan <- status
		}
	}()

	// Wait for either webhook notification or polling to complete
	select {
	case status := <-c.statusChan:
		log.Printf("Received status: %s\n", status)
		return status, nil
	case <-time.After(30 * time.Second):
		return c.GetLatestStatus(), fmt.Errorf("timeout waiting for job completion")
	}
}

// GetLatestStatus returns the latest known status of the job
func (c *Client) GetLatestStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.latestStatus
}
