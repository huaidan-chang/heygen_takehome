package client

import (
	"log"
	"testing"
)

func TestClientLibrary(t *testing.T) {
	client := NewClient("http://localhost:8080", "9090")

	status, err := client.StartClientWithWebhook()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if status == "completed" || status == "error" {
		log.Printf("Test passed: Job completed with status: %s", status)
	} else {
		t.Fatalf("Unexpected final status: %s", status)
	}
}
