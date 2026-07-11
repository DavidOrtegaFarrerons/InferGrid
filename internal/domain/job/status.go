package job

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusRunning   Status = "RUNNING"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusSucceeded, StatusFailed:
		return true
	default:
		return false
	}
}
