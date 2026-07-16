package job

import "testing"

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status string
		valid  bool
	}{
		{name: "Status PENDING is valid", status: string(StatusPending), valid: true},
		{name: "Status RUNNING is valid", status: string(StatusRunning), valid: true},
		{name: "Status SUCCEEDED is valid", status: string(StatusSucceeded), valid: true},
		{name: "Status FAILED is valid", status: string(StatusFailed), valid: true},
		{name: "Status FAKE is invalid", status: "FAKE", valid: false},
		{name: "Empty status is invalid", status: "", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Status(tt.status).IsValid()
			if got != tt.valid {
				t.Errorf("Status(%q).IsValid() = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}
