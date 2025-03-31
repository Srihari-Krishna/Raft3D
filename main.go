package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var node *RaftNode

func main() {
	fmt.Println("Starting Raft3D Node...")

	// Parse arguments
	isBootstrap := false
	var nodeID, dataDir, raftAddr, httpPort string

	if len(os.Args) > 1 {
		if os.Args[1] == "--bootstrap" {
			isBootstrap = true
			nodeID = "leader"
			dataDir = "./data/leader"
			raftAddr = "127.0.0.1:9001"
			httpPort = "8080"
		} else if os.Args[1] == "--join" && len(os.Args) > 2 {
			nodeID = os.Args[2]
			if nodeID == "node2" {
				dataDir = "./data/node2"
				raftAddr = "127.0.0.1:9002"
				httpPort = "8081"
			} else if nodeID == "node3" {
				dataDir = "./data/node3"
				raftAddr = "127.0.0.1:9003"
				httpPort = "8082"
			} else {
				log.Fatal("Unknown node ID. Use: node2, node3, etc.")
			}
		} else {
			log.Fatal("Usage: go run . --bootstrap OR go run . --join <nodeID>")
		}
	} else {
		log.Fatal("Missing arguments")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize Raft node
	var err error
	node, err = NewRaftNode(nodeID, dataDir, raftAddr, isBootstrap)
	if err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	fmt.Println("Raft node started successfully!")

	// Start HTTP server
	go func() {
		addr := ":" + httpPort
		fmt.Println("HTTP server listening on", addr)
		http.HandleFunc("/join", handleJoin)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	http.HandleFunc("/raft/state", handleRaftState)

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
