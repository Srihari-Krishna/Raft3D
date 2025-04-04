package main

import (
	"encoding/json"
	"io"
	"sync"
	"log"
	"github.com/hashicorp/raft"
)

// PrintJob represents a single 3D printing job
type PrintJob struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

// Printer represents a 3D printer in the system
type Printer struct {
	ID      string `json:"id"`
	Company string `json:"company"`
	Model   string `json:"model"`
}

// Filament represents a roll of filament
type Filament struct {
	ID                     string `json:"id"`
	Type                   string `json:"type"`
	Color                  string `json:"color"`
	TotalWeightInGrams     int    `json:"total_weight_in_grams"`
	RemainingWeightInGrams int    `json:"remaining_weight_in_grams"`
}

// RaftFSM is the FSM for storing printer, filament, and print job states
type RaftFSM struct {
	mu       sync.Mutex
	printers map[string]Printer
	filaments map[string]Filament
	jobs     map[string]PrintJob
}

// NewRaftFSM creates a new FSM
func NewRaftFSM() *RaftFSM {
	return &RaftFSM{
		printers:  make(map[string]Printer),
		filaments: make(map[string]Filament),
		jobs:      make(map[string]PrintJob),
	}
}

type PrintJobStatusUpdate struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// Apply applies a Raft log entry to the FSM
func (f *RaftFSM) Apply(logEntry *raft.Log) interface{} {
    f.mu.Lock()
    defer f.mu.Unlock()

    var data map[string]interface{}
    if err := json.Unmarshal(logEntry.Data, &data); err != nil {
        log.Println("RaftFSM Apply: Failed to unmarshal JSON into map:", err)
        return err
    }

    // ✅ Ensure new print jobs get added instead of being ignored
    if jobID, ok := data["job_id"].(string); ok {
        if _, exists := f.jobs[jobID]; !exists {
            // This is a new job, add it
            var job PrintJob
            if err := json.Unmarshal(logEntry.Data, &job); err != nil {
                log.Println("RaftFSM Apply: Failed to unmarshal PrintJob:", err)
                return err
            }
            f.jobs[job.JobID] = job
            log.Println(" Added new print job:", job)
        } else {
            // Existing job -> Update its status
            f.jobs[jobID] = PrintJob{
                JobID:   jobID,
                Status:  data["status"].(string),
                Details: f.jobs[jobID].Details, // Preserve existing details
            }
            log.Println(" Updated job status:", f.jobs[jobID])
        }
        return nil
    }

    if _, ok := data["company"]; ok {
        var printer Printer
        if err := json.Unmarshal(logEntry.Data, &printer); err != nil {
            log.Println("RaftFSM Apply: Failed to unmarshal Printer:", err)
            return err
        }
        f.printers[printer.ID] = printer
        log.Println(" Added printer:", printer)
    } else if _, ok := data["type"]; ok {
        var filament Filament
        if err := json.Unmarshal(logEntry.Data, &filament); err != nil {
            log.Println("RaftFSM Apply: Failed to unmarshal Filament:", err)
            return err
        }
        f.filaments[filament.ID] = filament
        log.Println(" Added filament:", filament)
    }

    return nil
}





// Snapshot takes a snapshot of the FSM state
func (f *RaftFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Copy state to snapshot
	snapshot := &raftFSMSnapshot{
		Printers:  make(map[string]Printer),
		Filaments: make(map[string]Filament),
		Jobs:      make(map[string]PrintJob),
	}

	// Copy each map
	for k, v := range f.printers {
		snapshot.Printers[k] = v
	}
	for k, v := range f.filaments {
		snapshot.Filaments[k] = v
	}
	for k, v := range f.jobs {
		snapshot.Jobs[k] = v
	}

	return snapshot, nil
}

// Restore restores FSM state from a snapshot
func (f *RaftFSM) Restore(reader io.ReadCloser) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	defer reader.Close()

	var snapshot raftFSMSnapshot
	if err := json.NewDecoder(reader).Decode(&snapshot); err != nil {
		return err
	}

	f.printers = snapshot.Printers
	f.filaments = snapshot.Filaments
	f.jobs = snapshot.Jobs

	return nil
}

// raftFSMSnapshot stores a snapshot of the FSM state
type raftFSMSnapshot struct {
	Printers  map[string]Printer  `json:"printers"`
	Filaments map[string]Filament `json:"filaments"`
	Jobs      map[string]PrintJob `json:"jobs"`
}


// Persist writes the snapshot to the given sink
func (s *raftFSMSnapshot) Persist(sink raft.SnapshotSink) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	if _, err := sink.Write(data); err != nil {
		_ = sink.Cancel()
		return err
	}

	return sink.Close()
}

// Release is a no-op (not needed here)
func (s *raftFSMSnapshot) Release() {}
