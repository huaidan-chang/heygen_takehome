package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
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
	}
}

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
