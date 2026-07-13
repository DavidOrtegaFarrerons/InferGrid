package stub

import "context"

type StubInferenceRunner struct {
}

func (s StubInferenceRunner) Generate(ctx context.Context, prompt string) (string, error) {
	return "this is a generated prompt by a stub inference runner", nil
}
