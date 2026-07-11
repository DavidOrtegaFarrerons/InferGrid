package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	inferencev1 "github.com/DavidOrtegaFarrerons/infergrid/proto/inference/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return usageError()
	}

	conn, err := grpc.NewClient(
		"localhost:9091",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connect to inference API: %w", err)
	}
	defer conn.Close()

	client := inferencev1.NewInferenceServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch os.Args[1] {
	case "submit":
		return runSubmit(ctx, client, os.Args[2:])
	case "get":
		return runGet(ctx, client, os.Args[2:])
	default:
		return usageError()
	}
}

func runSubmit(
	ctx context.Context,
	client inferencev1.InferenceServiceClient,
	args []string,
) error {
	prompt := strings.TrimSpace(strings.Join(args, " "))
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	resp, err := client.SubmitJob(
		ctx,
		&inferencev1.SubmitJobRequest{Prompt: prompt},
	)
	if err != nil {
		return fmt.Errorf("submit job: %w", err)
	}

	fmt.Printf("Job submitted: %s\n", resp.JobId)

	return nil
}

func runGet(
	ctx context.Context,
	client inferencev1.InferenceServiceClient,
	args []string,
) error {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("get requires a job ID")
	}

	resp, err := client.GetJob(
		ctx,
		&inferencev1.GetJobRequest{JobId: strings.TrimSpace(args[0])},
	)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	if resp.Job == nil {
		return fmt.Errorf("server returned an empty job")
	}

	printJob(resp.Job)

	return nil
}

func printJob(inferenceJob *inferencev1.Job) {
	fmt.Printf("ID: %s\n", inferenceJob.Id)
	fmt.Printf("Prompt: %s\n", inferenceJob.Prompt)
	fmt.Printf("Status: %s\n", inferenceJob.Status.String())
	fmt.Printf("Created: %s\n", inferenceJob.CreatedAt.AsTime().Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", inferenceJob.UpdatedAt.AsTime().Format(time.RFC3339))

	if inferenceJob.Result != nil {
		fmt.Printf("Result: %s\n", inferenceJob.GetResult())
	}

	if inferenceJob.FailureReason != nil {
		fmt.Printf("Failure: %s\n", inferenceJob.GetFailureReason())
	}
}

func usageError() error {
	return fmt.Errorf(
		"usage:\n  infergrid submit <prompt>\n  infergrid get <job-id>",
	)
}
