package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockModelsResponse(t *testing.T) {
	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the mock response function directly
	mockModelsResponse(rr)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the Content-Type header
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	// Parse the JSON response to verify its structure
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify the 'object' field is 'list' (OpenAI standard)
	if response["object"] != "list" {
		t.Errorf("Expected object to be 'list', got %v", response["object"])
	}

	// Verify data array is present
	data, ok := response["data"].([]interface{})
	if !ok || len(data) == 0 {
		t.Fatal("Expected 'data' array in response, but it was missing or empty")
	}
}