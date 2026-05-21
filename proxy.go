package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// ProxyResponse structures the JSON/JSONP wrapped output format
type ProxyResponse struct {
	Contents string `json:"contents"`
	Status   int    `json:"status"`
}

// ProxyHandler is the core engine for processing CORS proxy requests
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for frontend flexibility
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Extract the alias ID
	alias := r.URL.Query().Get("cros")
	if alias == "" {
		http.Error(w, "Bad Request: Missing 'cros' parameter", http.StatusBadRequest)
		return
	}

	// Fast O(1) lookup in the pre-loaded memory cache
	config, exists := ConfigMap[alias]
	if !exists {
		// Strict API Gateway Pattern: Deny unmapped requests immediately
		http.Error(w, "403 Forbidden: Alias mapping not found", http.StatusForbidden)
		return
	}

	// Initialize standard HTTP Client with a strict 10-second timeout constraint
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, config.TargetURL, nil)
	if err != nil {
		http.Error(w, "Internal Server Error: Failed to create request", http.StatusInternalServerError)
		return
	}

	// Set generic User-Agent to bypass rudimentary blocking mechanisms
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Execute request to the target
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Proxy Engine Error: Target %s unreachable: %v", config.TargetURL, err)
		http.Error(w, "502 Bad Gateway: Target server unreachable or timed out", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response payload
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "500 Internal Server Error: Failed to read target response", http.StatusInternalServerError)
		return
	}

	// Format response strictly based on config map specification
	switch config.Format {
	case "raw":
		// Direct Passthrough
		contentType := resp.Header.Get("Content-Type")
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(bodyBytes)

	case "json":
		// Standard JSON Wrapper
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		jsonResp := ProxyResponse{
			Contents: string(bodyBytes),
			Status:   resp.StatusCode,
		}
		json.NewEncoder(w).Encode(jsonResp)

	case "jsonp":
		// JSONP callback wrapper
		w.Header().Set("Content-Type", "application/javascript")
		w.WriteHeader(http.StatusOK)
		
		jsonResp := ProxyResponse{
			Contents: string(bodyBytes),
			Status:   resp.StatusCode,
		}
		jsonData, _ := json.Marshal(jsonResp)

		// Respect user's callback param or default to "callback"
		callback := r.URL.Query().Get("callback")
		if callback == "" {
			callback = "callback"
		}
		fmt.Fprintf(w, "%s(%s);", callback, string(jsonData))

	default:
		// Fallback for invalid configuration formats
		http.Error(w, "500 Internal Server Error: Invalid format in configuration", http.StatusInternalServerError)
	}
}
