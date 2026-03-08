package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// VERSION — edit this to verify hot reload is working.
const VERSION = "1.0.0"

// greeting — change this and save to trigger a reload.
const greeting = "Hello from the hot-reload test server!"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/hello", handleHello(logger))
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/version", handleVersion)

	addr := ":" + port
	logger.Info("Test server starting",
		"addr", addr,
		"version", VERSION,
		"started_at", time.Now().Format(time.RFC3339),
	)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>hotreload test server</title></head>
<body>
  <h1>hotreload test server</h1>
  <p>Version: <strong>%s</strong></p>
  <p>Started: %s</p>
  <ul>
    <li><a href="/hello">/hello</a></li>
    <li><a href="/version">/version</a></li>
    <li><a href="/health">/health</a></li>
  </ul>
  <p><em>Edit testserver/main.go and save — watch the terminal reload.</em></p>
</body>
</html>`, VERSION, time.Now().Format(time.RFC3339))
}

func handleHello(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Request", "path", r.URL.Path, "method", r.Method)
		fmt.Fprintln(w, greeting)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","version":"%s","time":"%s"}`,
		VERSION, time.Now().Format(time.RFC3339))
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, VERSION)
}
