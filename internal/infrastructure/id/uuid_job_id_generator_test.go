package id

import (
	"testing"

	"github.com/google/uuid"
)

func TestUuidJobIdGenerator_Generate(t *testing.T) {
	gen := NewUuidJobIdGenerator()

	first := gen.Generate()
	if first == "" {
		t.Fatal("Generate() returned an empty id")
	}
	if _, err := uuid.Parse(string(first)); err != nil {
		t.Errorf("Generate() = %q, not a valid UUID: %v", first, err)
	}

	if second := gen.Generate(); second == first {
		t.Errorf("Generate() returned duplicate ids: %q", first)
	}
}
