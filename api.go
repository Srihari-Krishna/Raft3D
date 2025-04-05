package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hashicorp/raft"
)

type APIHandler struct {
	raft *raft.Raft
	fsm  *RaftFSM
}

func NewAPIHandler(raftNode *raft.Raft, fsm *RaftFSM) *APIHandler {
	if fsm == nil {
		log.Fatal("RaftFSM is nil! Check initialization.")
	}
	return &APIHandler{
		raft: raftNode,
		fsm:  fsm,
	}
}

// Struct for job status update
type PrintJobStatusUpdate struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

func (h *APIHandler) createPrinter(w http.ResponseWriter, r *http.Request) {
	var printer Printer
	if err := json.NewDecoder(r.Body).Decode(&printer); err != nil {
		http.Error(w, "Invalid printer data", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(printer)
	if err != nil {
		http.Error(w, "Failed to encode printer data", http.StatusInternalServerError)
		return
	}

	if future := h.raft.Apply(data, 0); future.Error() != nil {
		http.Error(w, "Raft apply error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(printer)
}

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

func (h *APIHandler) createFilament(w http.ResponseWriter, r *http.Request) {
	var filament Filament
	if err := json.NewDecoder(r.Body).Decode(&filament); err != nil {
		http.Error(w, "Invalid filament data", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(filament)
	if err != nil {
		http.Error(w, "Failed to encode filament data", http.StatusInternalServerError)
		return
	}

	if future := h.raft.Apply(data, 0); future.Error() != nil {
		http.Error(w, "Raft apply error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(filament)
}

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

func (h *APIHandler) createPrintJob(w http.ResponseWriter, r *http.Request) {
	if h.raft.State() != raft.Leader {
		http.Error(w, "Not the leader", http.StatusBadRequest)
		return
	}

	var job PrintJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "Invalid print job data", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(job)
	if err != nil {
		http.Error(w, "Failed to encode print job", http.StatusInternalServerError)
		return
	}

	future := h.raft.Apply(data, 0)
	if err := future.Error(); err != nil {
		http.Error(w, "Raft apply error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 👇 Check FSM's response (e.g., filament not enough, job ID already exists, etc.)
	if respErr, ok := future.Response().(error); ok && respErr != nil {
		http.Error(w, respErr.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}



func (h *APIHandler) getPrintJobs(w http.ResponseWriter, r *http.Request) {
	h.fsm.mu.Lock()
	defer h.fsm.mu.Unlock()

	jobs := make([]PrintJob, 0, len(h.fsm.jobs))
	for _, job := range h.fsm.jobs {
		jobs = append(jobs, job)
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

	update := map[string]interface{}{
		"job_id": jobID,
		"status": status,
	}

	data, err := json.Marshal(update)
	if err != nil {
		http.Error(w, "Failed to encode status update", http.StatusInternalServerError)
		return
	}

	future := h.raft.Apply(data, 0)

	if future.Error() != nil {
		http.Error(w, future.Error().Error(), http.StatusInternalServerError)
		return
	}
	
	resp := future.Response()
	if err, ok := resp.(error); ok {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Print job status updated"})
}
