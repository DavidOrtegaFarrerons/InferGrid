package job

import (
	"strings"
	"time"
)

type ID string

type Job struct {
	id            ID
	prompt        string
	status        Status
	result        *string
	failureReason *string
	createdAt     time.Time
	updatedAt     time.Time
}

func New(id ID, prompt string) (*Job, error) {
	if id == "" {
		return nil, ErrEmptyID
	}

	if strings.TrimSpace(prompt) == "" {
		return nil, ErrEmptyPrompt
	}

	now := time.Now().UTC()

	return &Job{
		id:        id,
		prompt:    prompt,
		status:    StatusPending,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func Restore(
	id ID,
	prompt string,
	status Status,
	result *string,
	failureReason *string,
	createdAt time.Time,
	updatedAt time.Time,
) *Job {
	return &Job{
		id:            id,
		prompt:        prompt,
		status:        status,
		result:        result,
		failureReason: failureReason,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

func (j *Job) Start() error {
	if j.status != StatusPending {
		return ErrInvalidStatusTransition
	}

	j.status = StatusRunning
	j.updatedAt = time.Now().UTC()

	return nil
}

func (j *Job) Succeed(result string) error {
	if j.status != StatusRunning {
		return ErrInvalidStatusTransition
	}

	j.status = StatusSucceeded
	j.result = &result
	j.failureReason = nil
	j.updatedAt = time.Now().UTC()

	return nil
}

func (j *Job) Fail(reason string) error {
	if j.status != StatusRunning {
		return ErrInvalidStatusTransition
	}

	j.status = StatusFailed
	j.result = nil
	j.failureReason = &reason
	j.updatedAt = time.Now().UTC()

	return nil
}

func (j *Job) ID() ID {
	return j.id
}

func (j *Job) Prompt() string {
	return j.prompt
}

func (j *Job) Status() Status {
	return j.status
}

func (j *Job) Result() (string, bool) {
	if j.result == nil {
		return "", false
	}

	return *j.result, true
}

func (j *Job) FailureReason() (string, bool) {
	if j.failureReason == nil {
		return "", false
	}

	return *j.failureReason, true
}

func (j *Job) CreatedAt() time.Time {
	return j.createdAt
}

func (j *Job) UpdatedAt() time.Time {
	return j.updatedAt
}
