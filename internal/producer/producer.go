package producer

import (
	"encoding/json"
	"net/http"

	"job-queue/internal/queue"
)

// Enqueue accepts a JSON job and adds it to the queue.
func Enqueue(w http.ResponseWriter, r *http.Request, q *queue.Queue) {
	// Allow only POST requests.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	// Read the JSON job.
	var job queue.Job
	err := json.NewDecoder(r.Body).Decode(&job)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate the job.
	if job.ID <= 0 || !job.Type.Valid() || job.Payload == "" {
		http.Error(w, "Invalid job", http.StatusBadRequest)
		return
	}

	// Add the job to Redis.
	err = queue.Enqueue(r.Context(), q, job)
	if err != nil {
		http.Error(w, "Failed to enqueue job", http.StatusInternalServerError)
		return
	}

	// Send response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	json.NewEncoder(w).Encode(map[string]any{
		"status": "Job accepted",
		"id":     job.ID,
		"type":   job.Type,
	})
}
