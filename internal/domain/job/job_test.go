package job

import (
	"errors"
	"testing"
	"time"
)

func TestNewJob(t *testing.T) {
	tests := []struct {
		name           string
		id             ID
		prompt         string
		expectedError  error
		expectedStatus Status
	}{
		{
			name:          "empty id triggers error",
			id:            "",
			prompt:        "This is a cool prompt",
			expectedError: ErrEmptyID,
		},
		{
			name:          "empty prompt triggers error",
			id:            "ABCDEFG",
			prompt:        "",
			expectedError: ErrEmptyPrompt,
		},
		{
			name:          "empty prompt with spaces triggers error",
			id:            "ABCDEFG",
			prompt:        "           ",
			expectedError: ErrEmptyPrompt,
		},
		{
			name:           "status is new on new created job",
			id:             "ABCDEFG",
			prompt:         "cool prompt",
			expectedError:  nil,
			expectedStatus: StatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := New(tt.id, tt.prompt)
			if gotErr != nil {
				if !errors.Is(gotErr, tt.expectedError) {
					t.Errorf("Expected error %v, got %v", tt.expectedError, gotErr)
				}

				if got != nil {
					t.Errorf("Expected job to be nil because there has been an error, but got %v", got)
				}

				return
			}

			if got.prompt != tt.prompt {
				t.Errorf("Got prompt %s, expected %s", got.prompt, tt.prompt)
			}

			if got.id != tt.id {
				t.Errorf("Got id %s, expected %s", got.id, tt.id)
			}

			if got.status != tt.expectedStatus {
				t.Errorf("Got status %s, expected %s", got.status, tt.expectedStatus)
			}
		})
	}
}

func restoreWithStatus(status Status) *Job {
	past := time.Now().UTC().Add(-time.Hour)
	return Restore("id-1", "a prompt", status, nil, nil, past, past)
}

func TestJob_Start(t *testing.T) {
	tests := []struct {
		name       string
		initial    Status
		wantErr    error
		wantStatus Status
	}{
		{"pending can start", StatusPending, nil, StatusRunning},
		{"running cannot start again", StatusRunning, ErrInvalidStatusTransition, StatusRunning},
		{"succeeded cannot start", StatusSucceeded, ErrInvalidStatusTransition, StatusSucceeded},
		{"failed cannot start", StatusFailed, ErrInvalidStatusTransition, StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := restoreWithStatus(tt.initial)
			before := j.updatedAt

			err := j.Start()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Start() error = %v, want %v", err, tt.wantErr)
			}
			if j.status != tt.wantStatus {
				t.Errorf("status = %s, want %s", j.status, tt.wantStatus)
			}
			if tt.wantErr == nil && !j.updatedAt.After(before) {
				t.Errorf("expected updatedAt to advance on successful transition")
			}
		})
	}
}

func TestJob_Succeed(t *testing.T) {
	t.Run("running job succeeds and clears failure reason", func(t *testing.T) {
		reason := "an earlier failure"
		past := time.Now().UTC().Add(-time.Hour)
		j := Restore("id-1", "a prompt", StatusRunning, nil, &reason, past, past)

		if err := j.Succeed("the result"); err != nil {
			t.Fatalf("Succeed() unexpected error: %v", err)
		}
		if j.status != StatusSucceeded {
			t.Errorf("status = %s, want %s", j.status, StatusSucceeded)
		}
		result, ok := j.Result()
		if !ok || result != "the result" {
			t.Errorf("Result() = %q, %v; want %q, true", result, ok, "the result")
		}
		if _, ok := j.FailureReason(); ok {
			t.Errorf("expected failure reason to be cleared on success")
		}
	})

	for _, initial := range []Status{StatusPending, StatusSucceeded, StatusFailed} {
		t.Run("cannot succeed from "+string(initial), func(t *testing.T) {
			j := restoreWithStatus(initial)
			if err := j.Succeed("x"); !errors.Is(err, ErrInvalidStatusTransition) {
				t.Errorf("Succeed() error = %v, want %v", err, ErrInvalidStatusTransition)
			}
			if j.status != initial {
				t.Errorf("status changed to %s, want unchanged %s", j.status, initial)
			}
		})
	}
}

func TestJob_Fail(t *testing.T) {
	t.Run("running job fails and clears result", func(t *testing.T) {
		result := "an earlier result"
		past := time.Now().UTC().Add(-time.Hour)
		j := Restore("id-1", "a prompt", StatusRunning, &result, nil, past, past)

		if err := j.Fail("boom"); err != nil {
			t.Fatalf("Fail() unexpected error: %v", err)
		}
		if j.status != StatusFailed {
			t.Errorf("status = %s, want %s", j.status, StatusFailed)
		}
		reason, ok := j.FailureReason()
		if !ok || reason != "boom" {
			t.Errorf("FailureReason() = %q, %v; want %q, true", reason, ok, "boom")
		}
		if _, ok := j.Result(); ok {
			t.Errorf("expected result to be cleared on failure")
		}
	})

	for _, initial := range []Status{StatusPending, StatusSucceeded, StatusFailed} {
		t.Run("cannot fail from "+string(initial), func(t *testing.T) {
			j := restoreWithStatus(initial)
			if err := j.Fail("boom"); !errors.Is(err, ErrInvalidStatusTransition) {
				t.Errorf("Fail() error = %v, want %v", err, ErrInvalidStatusTransition)
			}
			if j.status != initial {
				t.Errorf("status changed to %s, want unchanged %s", j.status, initial)
			}
		})
	}
}

func TestJob_Getters_ZeroValues(t *testing.T) {
	j, err := New("id-1", "a prompt")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if got, ok := j.Result(); ok || got != "" {
		t.Errorf("Result() = %q, %v; want \"\", false", got, ok)
	}
	if got, ok := j.FailureReason(); ok || got != "" {
		t.Errorf("FailureReason() = %q, %v; want \"\", false", got, ok)
	}
}
