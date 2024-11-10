# HeyGen Take Home Project

## Project Architecture
The project is divided into two main components:

1. **Sever**(`server/server.go`): Simulates a video translation backend and provides two endpoints:
    - `/status`: Returns the job status (`pending`, `completed` or `error`).
    - `register-webhook`: Registers a webhook URL for job completion notification.
2. **Client**(`client/client.go`): A client library that interacts with the server.
    - Polls the server for job status.
    - Registers a webhook to receive job completion notifications.
    - Starts a webhook server to listen for completion notifications.
    - Uses an adaptive polling strategy to efficiently check for job completion.

The project also includes a test file (`client/client_test.go`) to test the hybrid polling and webhook system.

### Project Structure
```
heygen_takehome/
├── README.md
├── server/
│   └── server.go
├── client/
│   ├── client.go
│   └── client_test.go
```

## How to Start the Server
To start the server, navigate to the `server` directory and run the following command:
```sh
cd server
go run server.go
```
## How to Use the Client Library
### 1. Create a New Client
Create a new client by providing the server base URL and the port for the webhook server:
```go
client := NewClient("http://localhost:8080", "9090")
```
### 2. Start the Webhook Server
The `StartClientWithWebhook` method will automatically start the webhook server and register it with the main server.

### 3. Start Client with Webhook and Polling
You can use the `StartClientWithWebhook` function to start both the webhook and polling in a single method.
This function will return as soon as the job status is `"completed"` or `"error"` through either polling or webhook notification.

```go
status, err := client.StartClientWithWebhook()
if err != nil {
    log.Fatalf("Error starting client with webhook: %v", err)
}
log.Printf("Final Job Status: %s", status)
```

## How to Start the Test
To run the test, navigate to the `client` directory and use the `go test` command:
```sh
cd client
go test -v
```
The test will:
- Start a local webhook server.
- Register the webhook with the server.
- Start polling for the job status as a backup mechanism.
- Validate that the job completes via either polling or webhook notification.

## Explanation of the Approach
### Trivial Approach and Its Drawbacks
In a trivial approach, the client would repeatedly poll the server to get the job status. This can lead to two major issues:
- **Frequent Polling**: Polling too frequently can overwhelm the server, causing high CPU load, increased network traffic, and a potential decrease in server responsiveness.
- **Infrequent Polling**: Polling too infrequently can cause delays in receiving updates, resulting in a poor user experience as the client might not detect the job completion immediately.
### Hybrid Polling + Webhook Approach
The **Hybrid Polling + Webhook** approach offers a more efficient solution by combining polling with webhook notifications:

1. **Initial Polling**: The client starts by polling the server for updates, which triggers the start of the translation job.
2. **Webhook Registration**: During the initial polling phase, the client registers a webhook URL with the server. This webhook allows the server to notify the client immediately once the job is completed. This reduces the need for constant polling and helps in efficiently updating the client when the job is done.
3. **Adaptive Polling Strategy**: The polling strategy in the client library uses an adaptive approach where the interval between polling requests is adjusted based on the number of retries. Initially, the polling interval is longer (e.g., 3 seconds). As the client gets closer to reaching the maximum number of retries, the interval is reduced (e.g., 2 seconds, then 1 second). This ensures that the client polls more frequently as it expects the job to be near completion, minimizing the chances of missing a status change.
4. **Webhook as Primary Mechanism**: The webhook is designed to be the primary mechanism for notifying the client of job completion. The polling mechanism acts as a backup to ensure the client still gets the status if the webhook notification fails for any reason.
5. **Immediate Return on Notification**: The `StartClientWithWebhook` function returns the job status immediately upon receiving a webhook notification or when the polling result is "completed". This ensures that the client does not wait unnecessarily for the next polling cycle if either result is received earlier.
### Benefits Over a Trivial Approach
- **Reduced Server Load**: By relying on webhooks for the majority of the notifications, the server is not overwhelmed with constant polling requests.
- **Real-Time Updates**: Webhooks provide real-time notifications, ensuring that the client gets the job completion status as soon as it is ready.
- **Optimized Resource Usage**: The hybrid approach ensures that network resources are used efficiently, with polling used only when necessary and webhook notifications providing timely updates.

This hybrid strategy provides a balance between responsiveness and resource efficiency, making it an ideal solution for scenarios where jobs take a variable amount of time to complete.