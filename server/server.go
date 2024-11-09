package main

import (
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

const (
	Pending   = "pending"
	Completed = "completed"
	MinDelay  = 5
	MaxDelay  = 15
)

type jobRespStatus struct {
	Status string `json:"status"`
}

var (
	jobStatus       = Pending
	statusMutex     sync.Mutex
	jobStarted      = false
	jobStartedMutex sync.Mutex
	webhookURL      string
)

// statusHandler is the API to get the status of translation job
func statusHandler(w http.ResponseWriter, _ *http.Request) {
	statusMutex.Lock()
	defer statusMutex.Unlock()

	// Check if this is the first polling request and start the job timer if needed
	jobStartedMutex.Lock()
	if !jobStarted {
		jobStarted = true
		startJobCompletionTimer()
	}
	jobStartedMutex.Unlock()

	result := jobRespStatus{Status: jobStatus}
	err := json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// startJobCompletionTimer starts the timer for job completion after the first polling
func startJobCompletionTimer() {
	randomDelay := time.Duration(MinDelay+rand.IntN(MaxDelay-MinDelay+1)) * time.Second
	log.Printf("Job completion will be in: %v seconds\n", randomDelay.Seconds())

	go func() {
		time.Sleep(randomDelay)

		statusMutex.Lock()
		defer statusMutex.Unlock()

		jobStatus = Completed
		log.Println("Job completed")
	}()
}

// webhookRegisterHandler allows the client to register a webhook URL
func webhookRegisterHandler(w http.ResponseWriter, r *http.Request) {
	type webhookRequest struct {
		URL string `json:"url"`
	}

	var webhookReq webhookRequest
	err := json.NewDecoder(r.Body).Decode(&webhookReq)
	if err != nil {
		http.Error(w, "Invalid webhook request", http.StatusBadRequest)
		return
	}

	webhookURL = webhookReq.URL
	log.Printf("Registered webhook URL: %s\n", webhookURL)
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/register-webhook", webhookRegisterHandler)
	log.Println("Server is starting")
	// Server starts listening on port 8080
	log.Fatal(http.ListenAndServe(":8080", nil))
}
