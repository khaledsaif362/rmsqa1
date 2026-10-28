
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const defaultCommand = "/home/rsapp/Application/Exec/Run/ShowConnectStatus"
const commandDirectory = "/home/rsapp/Application/Exec/Run"

var commandRun sync.Mutex

type response struct {
	Status    string `json:"status"`
	Server    string `json:"server"`
	FetchedAt string `json:"fetched_at"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

func authorized(r *http.Request) bool {
	token := strings.TrimSpace(os.Getenv("MONITOR_TOKEN"))
	return token != "" &&
		r.Header.Get("Authorization") == "Bearer "+token
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !authorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	commandRun.Lock()
	defer commandRun.Unlock()

	command := strings.TrimSpace(os.Getenv("MARKET_STATUS_COMMAND"))
	if command == "" {
		command = defaultCommand
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", "cd /home/rsapp/Application/Exec/Run && ./ShowConnectStatus")
	cmd.Dir = commandDirectory

	output, err := cmd.CombinedOutput()
	host, _ := os.Hostname()

	result := response{
		Server:    host,
		FetchedAt: time.Now().Format(time.RFC3339),
		Output:    strings.TrimSpace(string(output)),
	}

	// The legacy executable can return exit code 1 while
	// still producing its exchange-status table.
	if !strings.Contains(result.Output, "Group Id") {
		result.Status = "error"
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Error = "ShowConnectStatus did not return a status table"
		}
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		result.Status = "success"
	}

	if e := json.NewEncoder(w).Encode(result); e != nil {
		log.Printf("encode response: %v", e)
	}
}

func main() {
	if strings.TrimSpace(os.Getenv("MONITOR_TOKEN")) == "" {
		log.Fatal("MONITOR_TOKEN is not set")
	}

	addr := strings.TrimSpace(os.Getenv("AGENT_LISTEN_ADDR"))
	if addr == "" {
		addr = "0.0.0.0:8095"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/monitoring/status", statusHandler)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("RMS monitoring API listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
