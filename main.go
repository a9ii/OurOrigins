package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// TargetConfig maps to the JSON object for each alias in config.json
type TargetConfig struct {
	TargetURL string `json:"target_url"`
	Format    string `json:"format"`
}

// ConfigMap acts as our global O(1) in-memory cache for routing lookups
var ConfigMap map[string]TargetConfig

// loadConfig reads and parses the configuration file strictly ONCE during startup.
func loadConfig(filename string) {
	// Read file from disk
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("FATAL: Failed to read config file '%s'. Error: %v", filename, err)
	}

	// Parse JSON into the memory map
	err = json.Unmarshal(data, &ConfigMap)
	if err != nil {
		log.Fatalf("FATAL: Failed to parse config JSON. Ensure valid syntax. Error: %v", err)
	}

	log.Printf("SUCCESS: Loaded %d alias targets from %s into memory cache.", len(ConfigMap), filename)
}

func main() {
	// 1. Initialize configuration before starting the HTTP server
	loadConfig("config.json")

	// 2. Route for serving the modern Frontend UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	// 3. Route for the core proxy API
	http.HandleFunc("/get", ProxyHandler)

	// 4. Start the server
	log.Println("Server is running. Access the UI at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
