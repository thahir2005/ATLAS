package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"
)

type Response struct {
	Service   string `json:"service"`
	Region    string `json:"region"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

var region string
var version = "1.0.0"

func respond(w http.ResponseWriter, status string) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(Response{
		Service:   "ATLAS Regional API",
		Region:    region,
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	healthcheck := flag.Bool("healthcheck", false, "Check local HTTP health endpoint")
	regionFlag := flag.String("region", "unknown", "Region name")
	flag.Parse()

	if *healthcheck {
		resp, err := http.Get("http://127.0.0.1:8080/health")
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("health check failed: %s", resp.Status)
		}
		return
	}

	region = *regionFlag

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		respond(w, "running")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respond(w, "healthy")
	})

	http.HandleFunc("/region", func(w http.ResponseWriter, r *http.Request) {
		respond(w, "available")
	})

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "ATLAS Regional API",
			"version": version,
			"region":  region,
		})
	})

	log.Printf("ATLAS Regional API starting: region=%s address=%s", region, *listen)

	server := &http.Server{
		Addr:              *listen,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
