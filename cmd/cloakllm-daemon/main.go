package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// RoutingPayload defines the structure sent to the external CGI proxy
type RoutingPayload struct {
	Endpoint string      `json:"endpoint"`
	Token    string      `json:"token"`
	Payload  interface{} `json:"payload"`
}

// SecureMessage is the AES-GCM encrypted wrapper
type SecureMessage struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// Global configuration variables loaded from the environment
var (
	SharedKey string
	ProxyURL  string
	TargetAPI string
	ModelsAPI string
)

// loadConfig reads and validates the required environment variables.
// It is explicitly called in main() to prevent it from running during unit tests.
func loadConfig() {
	SharedKey = os.Getenv("CLOAK_SHARED_KEY")
	ProxyURL = os.Getenv("CLOAK_PROXY_URL")
	TargetAPI = os.Getenv("CLOAK_TARGET_API")
	ModelsAPI = os.Getenv("CLOAK_MODELS_API") // Optional

	// Validate essential variables to prevent silent failures
	if SharedKey == "" || ProxyURL == "" || TargetAPI == "" {
		log.Fatal("[FATAL] CLOAK_SHARED_KEY, CLOAK_PROXY_URL, and CLOAK_TARGET_API must be set in the environment.")
	}
}

// handleProxyRequest intercepts the local request, encrypts it, and forwards it to the remote proxy.
func handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the token from the Authorization header provided by the frontend (e.g., Open WebUI)
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Handle GET requests for model endpoints
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models") {
		if ModelsAPI == "" {
			// If no upstream models API is defined, mock a default response to satisfy the frontend UI
			mockModelsResponse(w)
			return
		}
		// If an upstream API is defined, the request continues and is routed through the secure tunnel
	}

	// 2. Read the original request body (the LLM prompt)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Failed to read request body"}`, http.StatusBadRequest)
		return
	}

	var llmPayload map[string]interface{}
	// Only parse JSON if the body is not empty (GET requests usually have no body)
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &llmPayload); err != nil {
			http.Error(w, `{"error": "Invalid JSON payload"}`, http.StatusBadRequest)
			return
		}

		// FORCING non-streaming mode:
		// We override the 'stream' parameter to false to ensure a simple request-response 
		// cycle, which is required by the current proxy architecture.
    	llmPayload["stream"] = false
	}

	// 3. Determine the correct target endpoint
	upstreamEndpoint := TargetAPI
	if strings.HasSuffix(r.URL.Path, "/models") {
		upstreamEndpoint = ModelsAPI
	}

	// 4. Assemble the routing payload
	routingData := RoutingPayload{
		Endpoint: upstreamEndpoint,
		Token:    token,
		Payload:  llmPayload,
	}

	routingBytes, _ := json.Marshal(routingData)

	// 5. Encrypt the payload using the shared AES-GCM key
	nonceHex, ciphertextHex, err := EncryptPayload(routingBytes, SharedKey)
	if err != nil {
		http.Error(w, `{"error": "Encryption failed"}`, http.StatusInternalServerError)
		return
	}

	// 6. Wrap the encrypted data in the secure message structure
	secureReqBody, _ := json.Marshal(SecureMessage{
		Nonce:      nonceHex,
		Ciphertext: ciphertextHex,
	})

	// 7. Send the secure message through the TLS tunnel to the remote server proxy
	resp, err := http.Post(ProxyURL, "application/json", bytes.NewBuffer(secureReqBody))
	if err != nil {
		http.Error(w, `{"error": "Failed to connect to the external proxy"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 8. Read the proxy's response
	respBytes, _ := io.ReadAll(resp.Body)
	var secureResp SecureMessage
	if err := json.Unmarshal(respBytes, &secureResp); err != nil {
		http.Error(w, `{"error": "Invalid response from the external proxy"}`, http.StatusBadGateway)
		return
	}

	// 9. Decrypt the response from the remote proxy
	decryptedBytes, err := DecryptPayload(secureResp.Ciphertext, secureResp.Nonce, SharedKey)
	if err != nil {
		http.Error(w, `{"error": "Decryption of proxy response failed"}`, http.StatusInternalServerError)
		return
	}

	// 10. Return the plaintext JSON response to the local frontend
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(decryptedBytes)
}

// mockModelsResponse provides a static OpenAI-compatible response.
// This is used if CLOAK_MODELS_API is not set, ensuring tools like Open WebUI still function correctly.
func mockModelsResponse(w http.ResponseWriter) {
	mockResponse := `
	{
		"object": "list",
		"data": [
			{
				"id": "cloak-stealth-model",
				"object": "model",
				"created": 1686935002,
				"owned_by": "cloakllm"
			}
		]
	}`
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(mockResponse))
}

func main() {
	// 1. Explicitly load configuration on startup
	loadConfig()

	// 2. Register the local HTTP endpoints
	http.HandleFunc("/v1/chat/completions", handleProxyRequest)
	http.HandleFunc("/v1/models", handleProxyRequest)

	port := ":8080"
	fmt.Printf("[CloakLLM] Daemon initialized and listening on http://localhost%s\n", port)
	fmt.Printf("[CloakLLM] Routing traffic securely to %s\n", ProxyURL)
	
	// 3. Start the daemon
	log.Fatal(http.ListenAndServe(port, nil))
}