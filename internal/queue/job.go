package queue

// JobType identifies the kind of work a job represents.
type JobType string

const (
	SendEmail   JobType = "send_email"
	ResizeImage JobType = "resize_image"
	GeneratePDF JobType = "generate_pdf"
)

// Validating  job
func (t JobType) Valid() bool {
	switch t {
	case SendEmail, ResizeImage, GeneratePDF:
		return true
	default:
		return false
	}
}

// Job is a single item in the queue.
type Job struct {
	ID      int     `json:"id"`
	Type    JobType `json:"type"`
	Payload string  `json:"payload"`
}
