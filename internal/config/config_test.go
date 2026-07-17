package config

import (
	"testing"
)

// setWorkerBaseEnv sets every env var the API/worker share to valid values.
func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("MIGRATIONS_PATH", "file://some/path")
	t.Setenv("AMQP_URL", "amqp://user:pass@localhost:5672/")
	t.Setenv("GRPC_LISTEN_ADDRESS", ":9091")
}

func TestLoadClient(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Setenv("GRPC_LISTEN_ADDRESS", ":9091")

		cfg, err := LoadClient()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Server.ListenAddress != ":9091" {
			t.Errorf("ListenAddress = %q, want %q", cfg.Server.ListenAddress, ":9091")
		}
	})

	t.Run("missing address errors", func(t *testing.T) {
		t.Setenv("GRPC_LISTEN_ADDRESS", "")

		if _, err := LoadClient(); err == nil {
			t.Fatal("expected error when GRPC_LISTEN_ADDRESS is missing")
		}
	})
}

func TestLoadAPI(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		setBaseEnv(t)

		cfg, err := LoadAPI()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Database.DSN == "" || cfg.RabbitMQ.AMQPURL == "" || cfg.Server.ListenAddress != ":9091" {
			t.Errorf("unexpected config: %+v", cfg)
		}
	})

	t.Run("missing DATABASE_DSN errors", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("DATABASE_DSN", "")

		if _, err := LoadAPI(); err == nil {
			t.Fatal("expected error when DATABASE_DSN is missing")
		}
	})

	t.Run("migrations path falls back to default", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("MIGRATIONS_PATH", "")

		cfg, err := LoadAPI()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "file://internal/infrastructure/postgres/migrations"
		if cfg.Database.MigrationsPath != want {
			t.Errorf("MigrationsPath = %q, want default %q", cfg.Database.MigrationsPath, want)
		}
	})
}

func TestLoadWorker(t *testing.T) {
	t.Run("ollama provider loads ollama config", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("INFERENCE_PROVIDER", "ollama")
		t.Setenv("OLLAMA_URL", "http://localhost:11434")
		t.Setenv("OLLAMA_MODEL", "llama3.2:1b")

		cfg, err := LoadWorker()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Provider != ProviderOllama {
			t.Errorf("provider = %q, want %q", cfg.Provider, ProviderOllama)
		}
		if cfg.Ollama.Model != "llama3.2:1b" || cfg.Ollama.BaseURL != "http://localhost:11434" {
			t.Errorf("unexpected ollama config: %+v", cfg.Ollama)
		}
	})

	t.Run("openai-compatible provider loads its config", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("INFERENCE_PROVIDER", "openai-compatible")
		t.Setenv("OPENAI_COMPATIBLE_URL", "https://api.example.com")
		t.Setenv("OPENAI_COMPATIBLE_MODEL", "gpt-x")
		t.Setenv("OPENAI_COMPATIBLE_API_KEY", "secret")

		cfg, err := LoadWorker()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Provider != ProviderOpenAICompatible {
			t.Errorf("provider = %q, want %q", cfg.Provider, ProviderOpenAICompatible)
		}
		if cfg.OpenAICompatible.Model != "gpt-x" || cfg.OpenAICompatible.APIKey != "secret" {
			t.Errorf("unexpected openai config: %+v", cfg.OpenAICompatible)
		}
	})

	t.Run("openai-compatible works without an api key", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("INFERENCE_PROVIDER", "openai-compatible")
		t.Setenv("OPENAI_COMPATIBLE_URL", "https://api.example.com")
		t.Setenv("OPENAI_COMPATIBLE_MODEL", "gpt-x")
		t.Setenv("OPENAI_COMPATIBLE_API_KEY", "")

		cfg, err := LoadWorker()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.OpenAICompatible.APIKey != "" {
			t.Errorf("APIKey = %q, want empty", cfg.OpenAICompatible.APIKey)
		}
	})

	t.Run("unknown provider errors", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("INFERENCE_PROVIDER", "banana")

		if _, err := LoadWorker(); err == nil {
			t.Fatal("expected error for unknown INFERENCE_PROVIDER")
		}
	})

	t.Run("missing ollama model errors", func(t *testing.T) {
		setBaseEnv(t)
		t.Setenv("INFERENCE_PROVIDER", "ollama")
		t.Setenv("OLLAMA_URL", "http://localhost:11434")
		t.Setenv("OLLAMA_MODEL", "")

		if _, err := LoadWorker(); err == nil {
			t.Fatal("expected error when OLLAMA_MODEL is missing")
		}
	})
}
