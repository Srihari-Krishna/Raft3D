package main

import (
	"encoding/json"
	"net/http"
	"fmt"
	"log"
	"github.com/gorilla/mux"
	"github.com/hashicorp/raft"
)

// APIHandler manages HTTP requests
type APIHandler struct {
	raft *raft.Raft
	fsm  *RaftFSM
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(raftNode *raft.Raft, fsm *RaftFSM) *APIHandler {
    if fsm == nil {
        log.Fatal("RaftFSM is nil! Check initialization.")
    }
    return &APIHandler{
        raft: raftNode,
        fsm:  fsm,
    }
}


// Handle creating a new printer
func (h *APIHandler) createPrinter(w http.ResponseWriter, r *http.Request) {
	var printer Printer
	if err := json.NewDecoder(r.Body).Decode(&printer); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Convert printer to JSON for Raft log entry
	data, err := json.Marshal(printer)
	if err != nil {
		http.Error(w, "Failed to process data", http.StatusInternalServerError)
		return
	}

	// Apply to Raft log
	future := h.raft.Apply(data, 0)
	if err := future.Error(); err != nil {
		http.Error(w, "Raft apply error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(printer)
}

// Handle retrieving all printers
func (h *APIHandler) getPrinters(w http.ResponseWriter, r *http.Request) {
	h.fsm.mu.Lock()
	defer h.fsm.mu.Unlock()

	printers := make([]Printer, 0, len(h.fsm.printers))
	for _, p := range h.fsm.printers {
		printers = append(printers, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(printers)
}

// Handle creating a new filament
func (h *APIHandler) createFilament(w http.ResponseWriter, r *http.Request) {
	var filament Filament
	if err := json.NewDecoder(r.Body).Decode(&filament); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(filament)
	if err != nil {
		http.Error(w, "Failed to process data", http.StatusInternalServerError)
		return
	}

	future := h.raft.Apply(data, 0)
	if err := future.Error(); err != nil {
		http.Error(w, "Raft apply error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(filament)
}

// Handle retrieving all filaments
func (h *APIHandler) getFilaments(w http.ResponseWriter, r *http.Request) {
	h.fsm.mu.Lock()
	defer h.fsm.mu.Unlock()

	filaments := make([]Filament, 0, len(h.fsm.filaments))
	for _, f := range h.fsm.filaments {
		filaments = append(filaments, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filaments)
}

// Handle creating a new print job
func (h *APIHandler) createPrintJob(w http.ResponseWriter, r *http.Request) {
	var job PrintJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(job)
	if err != nil {
		http.Error(w, "Failed to process data", http.StatusInternalServerError)
		return
	}

	future := h.raft.Apply(data, 0)
	if err := future.Error(); err != nil {
		http.Error(w, "Raft apply error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

// Handle retrieving all print jobs
func (h *APIHandler) getPrintJobs(w http.ResponseWriter, r *http.Request) {
    if h.fsm == nil {
        http.Error(w, "Internal server error: FSM is nil", http.StatusInternalServerError)
        return
    }

    h.fsm.mu.Lock()
    defer h.fsm.mu.Unlock()

    if h.fsm.jobs == nil {
        http.Error(w, "No print jobs available", http.StatusNotFound)
        return
    }

    fmt.Println("Number of print jobs:", len(h.fsm.jobs)) // Debugging

    jobs := make([]PrintJob, 0, len(h.fsm.jobs))
    for _, j := range h.fsm.jobs {
        jobs = append(jobs, j)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(jobs)
}


func (h *APIHandler) updatePrintJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := mux.Vars(r)["job_id"]
	status := r.URL.Query().Get("status")

	if jobID == "" || status == "" {
		http.Error(w, "Missing job_id or status", http.StatusBadRequest)
		return
	}

	update := PrintJobStatusUpdate{JobID: jobID, Status: status}
	data, err := json.Marshal(update)
	if err != nil {
		http.Error(w, "Failed to process data", http.StatusInternalServerError)
		return
	}

	// Apply to Raft log
	future := h.raft.Apply(data, 0)
	if err := future.Error(); err != nil {
		http.Error(w, "Raft apply error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Print job status updated"})
}


