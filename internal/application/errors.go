package application

import "errors"

var (
	ErrJobNotFound          = errors.New("job not found")
	ErrInferenceUnavailable = errors.New("inference provider unavailable")
)
