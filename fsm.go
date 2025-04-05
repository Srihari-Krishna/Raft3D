package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/hashicorp/raft"
)

type PrintJob struct {
	JobID              string `json:"job_id"`
	PrinterID          string `json:"printer_id"`
	FilamentID         string `json:"filament_id"`
	PrintWeightInGrams int    `json:"print_weight_in_grams"`
	Status             string `json:"status"`
	Details            string `json:"details"`
}

type Printer struct {
	ID      string `json:"id"`
	Company string `json:"company"`
	Model   string `json:"model"`
}

type Filament struct {
	ID                     string `json:"id"`
	Type                   string `json:"type"`
	Color                  string `json:"color"`
	TotalWeightInGrams     int    `json:"total_weight_in_grams"`
	RemainingWeightInGrams int    `json:"remaining_weight_in_grams"`
}

type RaftFSM struct {
	mu        sync.Mutex
	printers  map[string]Printer
	filaments map[string]Filament
	jobs      map[string]PrintJob
}

func NewRaftFSM() *RaftFSM {
	return &RaftFSM{
		printers:  make(map[string]Printer),
		filaments: make(map[string]Filament),
		jobs:      make(map[string]PrintJob),
	}
}

func (f *RaftFSM) Apply(logEntry *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var data map[string]interface{}
	if err := json.Unmarshal(logEntry.Data, &data); err != nil {
		log.Println("Apply: Failed to unmarshal log:", err)
		return err
	}

	jobID, hasJobID := data["job_id"].(string)
	statusStr, hasStatus := data["status"].(string)

	// -----------------------------------------
	// Status update for existing PrintJob
	// -----------------------------------------
	if hasJobID && hasStatus && statusStr != "Queued" {
		job, exists := f.jobs[jobID]
		if !exists {
			return fmt.Errorf("job %s not found", jobID)
		}

		if !isValidTransition(job.Status, statusStr) {
			return fmt.Errorf("invalid transition from %s to %s", job.Status, statusStr)
		}

		if statusStr == "Done" {
			filament := f.filaments[job.FilamentID]
			filament.RemainingWeightInGrams -= job.PrintWeightInGrams
			f.filaments[job.FilamentID] = filament
		}

		job.Status = statusStr
		f.jobs[jobID] = job
		log.Println("Updated job:", job)
		return nil
	}

	// -----------------------------------------
	// Create new PrintJob
	// -----------------------------------------
	if hasJobID && data["printer_id"] != nil && data["filament_id"] != nil && data["print_weight_in_grams"] != nil {
		var job PrintJob
		if err := json.Unmarshal(logEntry.Data, &job); err != nil {
			log.Println("Apply: Invalid PrintJob:", err)
			return err
		}

		if _, exists := f.jobs[job.JobID]; exists {
			return fmt.Errorf("job with ID %s already exists", job.JobID)
		}

		if _, ok := f.printers[job.PrinterID]; !ok {
			return fmt.Errorf("invalid printer ID: %s", job.PrinterID)
		}

		filament, ok := f.filaments[job.FilamentID]
		if !ok {
			return fmt.Errorf("invalid filament ID: %s", job.FilamentID)
		}

		queuedWeight := f.calculateQueuedFilamentUsage(job.FilamentID)
		if job.PrintWeightInGrams > (filament.RemainingWeightInGrams - queuedWeight) {
			return fmt.Errorf("not enough filament available")
		}

		// Override whatever came with "status" — we control it.
		job.Status = "Queued"
		f.jobs[job.JobID] = job
		log.Println("New job added:", job)
		return nil
	}

	// -----------------------------------------
	// Create Printer
	// -----------------------------------------
	if _, ok := data["company"]; ok && data["model"] != nil && data["id"] != nil {
		var printer Printer
		if err := json.Unmarshal(logEntry.Data, &printer); err != nil {
			log.Println("Apply: Invalid Printer:", err)
			return err
		}
		f.printers[printer.ID] = printer
		log.Println("Printer added:", printer)
		return nil
	}

	// -----------------------------------------
	// Create Filament
	// -----------------------------------------
	if _, ok := data["type"]; ok && data["id"] != nil {
		var filament Filament
		if err := json.Unmarshal(logEntry.Data, &filament); err != nil {
			log.Println("Apply: Invalid Filament:", err)
			return err
		}
		f.filaments[filament.ID] = filament
		log.Println("Filament added:", filament)
		return nil
	}

	return fmt.Errorf("unknown data type or malformed input")
}



func isValidTransition(oldStatus, newStatus string) bool {
	transitions := map[string][]string{
		"Queued":   {"Running", "Cancelled"},
		"Running":  {"Done", "Cancelled"},
		"Done":     {},
		"Cancelled": {},
	}
	for _, valid := range transitions[oldStatus] {
		if newStatus == valid {
			return true
		}
	}
	return false
}

func (f *RaftFSM) calculateQueuedFilamentUsage(filamentID string) int {
	total := 0
	for _, job := range f.jobs {
		if job.FilamentID == filamentID && (job.Status == "Queued" || job.Status == "Running") {
			total += job.PrintWeightInGrams
		}
	}
	return total
}

func (f *RaftFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	snapshot := &raftFSMSnapshot{
		Printers:  copyPrinters(f.printers),
		Filaments: copyFilaments(f.filaments),
		Jobs:      copyJobs(f.jobs),
	}
	return snapshot, nil
}

func (f *RaftFSM) Restore(reader io.ReadCloser) error {
	defer reader.Close()
	f.mu.Lock()
	defer f.mu.Unlock()

	var snapshot raftFSMSnapshot
	if err := json.NewDecoder(reader).Decode(&snapshot); err != nil {
		return err
	}
	f.printers = snapshot.Printers
	f.filaments = snapshot.Filaments
	f.jobs = snapshot.Jobs
	return nil
}

type raftFSMSnapshot struct {
	Printers  map[string]Printer  `json:"printers"`
	Filaments map[string]Filament `json:"filaments"`
	Jobs      map[string]PrintJob `json:"jobs"`
}

func (s *raftFSMSnapshot) Persist(sink raft.SnapshotSink) error {
	data, err := json.Marshal(s)
	if err != nil {
		_ = sink.Cancel()
		return err
	}
	if _, err := sink.Write(data); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *raftFSMSnapshot) Release() {}

func copyPrinters(src map[string]Printer) map[string]Printer {
	dst := make(map[string]Printer)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyFilaments(src map[string]Filament) map[string]Filament {
	dst := make(map[string]Filament)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyJobs(src map[string]PrintJob) map[string]PrintJob {
	dst := make(map[string]PrintJob)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
