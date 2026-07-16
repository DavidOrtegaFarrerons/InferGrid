package config

import (
	"fmt"
	"log"
	"os"
)

type DatabaseConfig struct {
	DSN            string
	MigrationsPath string
}

type RabbitMQConfig struct {
	AMQPURL string
}

type GRPCServerConfig struct {
	ListenAddress string
}

type InferenceProvider string

const (
	ProviderOllama InferenceProvider = "ollama"
	ProviderOpenAICompatible InferenceProvider = "openai-compatible"
)

type OllamaConfig struct {
	BaseURL string
	Model   string
}

type OpenAICompatibleConfig struct {
	BaseURL string
	Model string
	APIKey string
}

type APIConfig struct {
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Server   GRPCServerConfig
}

type WorkerConfig struct {
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Server   GRPCServerConfig
	Provider InferenceProvider
	Ollama   OllamaConfig
	OpenAICompatible OpenAICompatibleConfig
}

type ClientConfig struct {
	Server GRPCServerConfig
}

func requiredEnv(key string) (string, error) {
	envVar := os.Getenv(key)
	if envVar == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return envVar, nil
}

func loadDatabase() (DatabaseConfig, error) {
	dsn, err := requiredEnv("DATABASE_DSN")
	if err != nil {
		return DatabaseConfig{}, err
	}

	migrationsPath, _ := requiredEnv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://internal/infrastructure/postgres/migrations"
	}

	return DatabaseConfig{
		DSN:            dsn,
		MigrationsPath: migrationsPath,
	}, nil
}

func loadRabbitMQ() (RabbitMQConfig, error) {
	amqpURL, err := requiredEnv("AMQP_URL")
	if err != nil {
		return RabbitMQConfig{}, err
	}

	return RabbitMQConfig{AMQPURL: amqpURL}, nil
}

func loadGRPCServer() (GRPCServerConfig, error) {
	listenAddress, err := requiredEnv("GRPC_LISTEN_ADDRESS")
	if err != nil {
		return GRPCServerConfig{}, err
	}

	return GRPCServerConfig{ListenAddress: listenAddress}, nil
}

func loadInferenceProvider() (InferenceProvider, error) {
	inferenceProvider, err := requiredEnv("INFERENCE_PROVIDER")
	if err != nil {
		return "", err
	}

	switch InferenceProvider(inferenceProvider) {
	case ProviderOllama, ProviderOpenAICompatible:
		return InferenceProvider(inferenceProvider), nil
	default:
		return "", fmt.Errorf("unknown INFERENCE_PROVIDER: %s", inferenceProvider)
	}
}

func loadOllama() (OllamaConfig, error) {
	url, err := requiredEnv("OLLAMA_URL")
	if err != nil {
		return OllamaConfig{}, err
	}

	model, err := requiredEnv("OLLAMA_MODEL")
	if err != nil {
		return OllamaConfig{}, err
	}

	return OllamaConfig{
		BaseURL: url,
		Model:   model,
	}, nil
}

func loadOpenAICompatible() (OpenAICompatibleConfig, error) {
	url, err := requiredEnv("OPENAI_COMPATIBLE_URL")
	if err != nil {
		return OpenAICompatibleConfig{}, err
	}

	model, err := requiredEnv("OPENAI_COMPATIBLE_MODEL")
	if err != nil {
		return OpenAICompatibleConfig{}, err
	}

	apiKey := os.Getenv("OPENAI_COMPATIBLE_API_KEY")
	if apiKey == "" {
		log.Println("No OPENAI_COMPATIBLE_API_KEY provided")
	}

	return OpenAICompatibleConfig{
		BaseURL: url,
		Model:   model,
		APIKey:  apiKey,
	}, nil
}

func LoadAPI() (APIConfig, error) {
	database, err := loadDatabase()
	if err != nil {
		return APIConfig{}, fmt.Errorf("loading database: %w", err)
	}

	rabbitMQ, err := loadRabbitMQ()
	if err != nil {
		return APIConfig{}, fmt.Errorf("loading rabbitmq: %w", err)
	}

	server, err := loadGRPCServer()
	if err != nil {
		return APIConfig{}, fmt.Errorf("loading server: %w", err)
	}

	return APIConfig{
		Database: database,
		RabbitMQ: rabbitMQ,
		Server:   server,
	}, nil
}

func LoadWorker() (WorkerConfig, error) {
	database, err := loadDatabase()
	if err != nil {
		return WorkerConfig{}, fmt.Errorf("loading database: %w", err)
	}

	rabbitMQ, err := loadRabbitMQ()
	if err != nil {
		return WorkerConfig{}, fmt.Errorf("loading rabbitmq: %w", err)
	}

	server, err := loadGRPCServer()
	if err != nil {
		return WorkerConfig{}, fmt.Errorf("loading server: %w", err)
	}

	provider, err := loadInferenceProvider()
	if err != nil {
		return WorkerConfig{}, fmt.Errorf("loading inference provider: %w", err)
	}

	cfg := WorkerConfig{
		Database: database,
		RabbitMQ: rabbitMQ,
		Server:   server,
		Provider: provider,
	}

	switch provider {
	case ProviderOllama:
		cfg.Ollama, err = loadOllama()
	case ProviderOpenAICompatible:
		cfg.OpenAICompatible,  err = loadOpenAICompatible()
	}
	if err != nil {
		return WorkerConfig{}, fmt.Errorf("loading %s config: %w", provider, err)
	}

	return cfg, nil

}

func LoadClient() (ClientConfig, error) {
	server, err := loadGRPCServer()
	if err != nil {
		return ClientConfig{}, fmt.Errorf("loading server: %w", err)
	}

	return ClientConfig{Server: server}, nil
}
