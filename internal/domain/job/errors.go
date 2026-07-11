package job

import "errors"

var (
	ErrEmptyID                 = errors.New("ID cannot be empty")
	ErrEmptyPrompt             = errors.New("prompt cannot be empty")
	ErrInvalidStatusTransition = errors.New("the status current status does not allow this transition")
)
