package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/gorilla/mux"
)

var node *RaftNode

func main() {
	fmt.Println("Starting Raft3D Node...")

	// Define command-line flags
	isBootstrap := flag.Bool("bootstrap", false, "Start as bootstrap node")
	nodeID := flag.String("node_id", "", "Unique node ID")
	raftAddr := flag.String("raft_addr", "", "Raft network address (host:port)")
	dataDir := flag.String("data_dir", "", "Directory for Raft data")
	httpPort := flag.String("http_port", "", "HTTP API port")

	flag.Parse()

	// Validate inputs
	if *isBootstrap {
		if *nodeID == "" {
			*nodeID = "leader"
		}
		if *raftAddr == "" {
			*raftAddr = "127.0.0.1:9001"
		}
		if *dataDir == "" {
			*dataDir = "./data/leader"
		}
		if *httpPort == "" {
			*httpPort = "8081"
		}
	} else {
		if *nodeID == "" || *raftAddr == "" || *dataDir == "" || *httpPort == "" {
			log.Fatal("Usage: go run . --bootstrap OR go run . --node_id=<ID> --raft_addr=<host:port> --data_dir=<dir> --http_port=<port>")
		}
	}

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize Raft node
	var err error
	node, err = NewRaftNode(*nodeID, *dataDir, *raftAddr, *isBootstrap)
	if err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	fmt.Println("Raft node started successfully!")

	// Start HTTP server
	go func() {
		router := mux.NewRouter()

		// Join & Raft State
		router.HandleFunc("/join", handleJoin).Methods("POST")
		router.HandleFunc("/raft/state", handleRaftState).Methods("GET")

		// Setup API routes
		handler := NewAPIHandler(node.raft, node.fsm)

		router.HandleFunc("/api/v1/printers", handler.createPrinter).Methods("POST")
		router.HandleFunc("/api/v1/printers", handler.getPrinters).Methods("GET")

		router.HandleFunc("/api/v1/filaments", handler.createFilament).Methods("POST")
		router.HandleFunc("/api/v1/filaments", handler.getFilaments).Methods("GET")

		router.HandleFunc("/api/v1/print_jobs", handler.createPrintJob).Methods("POST")
		router.HandleFunc("/api/v1/print_jobs", handler.getPrintJobs).Methods("GET")
		router.HandleFunc("/api/v1/print_jobs/{job_id}/status", handler.updatePrintJobStatus).Methods("PUT")

		addr := ":" + *httpPort
		fmt.Println("Unified HTTP server listening on", addr)
		if err := http.ListenAndServe(addr, router); err != nil {
			log.Fatalf("Unified HTTP server failed: %v", err)
		}
	}()

	// Keep the node running
	for {
		time.Sleep(10 * time.Second)
	}
}

// handleJoin handles incoming join requests
func handleJoin(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	addr := r.URL.Query().Get("addr")

	if nodeID == "" || addr == "" {
		http.Error(w, "Missing node_id or addr", http.StatusBadRequest)
		return
	}

	if err := node.JoinCluster(nodeID, addr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Node %s added successfully!", nodeID)
}

func handleRaftState(w http.ResponseWriter, r *http.Request) {
	state := node.raft.State().String()
	fmt.Fprintf(w, "Current state: %s", state)
}
