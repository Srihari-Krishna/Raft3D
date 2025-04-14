package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"strings"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/go-hclog"

)

// RaftNode represents a Raft node
type RaftNode struct {
	raft      *raft.Raft
	transport *raft.NetworkTransport
	config    *raft.Config
	fsm  	  *RaftFSM 
}

type filteredHCLLogger struct {
	hclog.Logger
}

func NewRaftNode(nodeID, dataDir, raftAddr string, isBootstrap bool) (*RaftNode, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)

	config.ElectionTimeout = 2 * time.Second  
	config.HeartbeatTimeout = 1 * time.Second  
	config.MaxAppendEntries = 10              
	config.TrailingLogs = 100      
	config.SnapshotInterval = 20 * time.Second 
	config.SnapshotThreshold = 10       
	baseLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "raft",
		Level: hclog.Info, 
	})
	config.Logger = filteredHCLLogger{Logger: baseLogger}
		

	logStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("%s/logs.bolt", dataDir))
	if err != nil {
		return nil, fmt.Errorf("failed to create log store: %v", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("%s/stable.bolt", dataDir))
	if err != nil {
		return nil, fmt.Errorf("failed to create stable store: %v", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(dataDir, 1, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot store: %v", err)
	}

	transport, err := raft.NewTCPTransport(raftAddr, nil, 3, 10*time.Second, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %v", err)
	}

	// ✅ Add FSM here
	fsm := NewRaftFSM()

	// ✅ Pass FSM to Raft
	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create Raft instance: %v", err)
	}

	if isBootstrap {
		fmt.Println("Bootstrapping Raft cluster...")
		future := raftNode.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(nodeID),
					Address: transport.LocalAddr(),
				},
			},
		})

		if err := future.Error(); err != nil {
			return nil, fmt.Errorf("failed to bootstrap cluster: %v", err)
		}
	} else {
		fmt.Printf("Follower node %s started at %s\n", nodeID, raftAddr)
	}

	return &RaftNode{
		raft:      raftNode,
		transport: transport,
		config:    config,
		fsm:       fsm,
	}, nil
}


// JoinCluster adds a new node to the Raft cluster
func (n *RaftNode) JoinCluster(nodeID, addr string) error {
	log.Printf("Joining new node %s at %s\n", nodeID, addr)

	if n.raft.State() != raft.Leader {
		return fmt.Errorf("only the leader can add nodes")
	}

	f := n.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := f.Error(); err != nil {
		return fmt.Errorf("failed to add node: %v", err)
	}

	log.Printf("Node %s added successfully!", nodeID)
	return nil
}

func (f filteredHCLLogger) Error(msg string, args ...interface{}) {
	if strings.Contains(msg, "Rollback failed: tx closed") {
		return // suppress this one
	}
	f.Logger.Error(msg, args...)
}

func (f filteredHCLLogger) Warn(msg string, args ...interface{}) {
	if strings.Contains(msg, "Rollback failed: tx closed") {
		return
	}
	f.Logger.Warn(msg, args...)
}
