package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"gtft-crawler/internal/config"
	"gtft-crawler/internal/fetcher"
	"gtft-crawler/internal/parser"
	"gtft-crawler/internal/storage"
	"gtft-crawler/internal/worker"
)

func main() {
	// Parse command line flags
	cfg := config.New()
	cfg.ParseFlags()

	fmt.Println("=== GTFT Academic Paper Crawler ===")
	fmt.Printf("Input file: %s\n", cfg.InputFile)
	fmt.Printf("Output directory: %s\n", cfg.OutputDir)
	fmt.Printf("Workers: %d\n", cfg.Workers)
	fmt.Printf("Rate limit: %d requests/second\n", cfg.RateLimit)
	fmt.Printf("Timeout: %v\n", cfg.Timeout)
	fmt.Printf("Max retries: %d\n", cfg.MaxRetries)
	fmt.Println()

	// Read URLs from file
	urls, err := readURLs(cfg.InputFile)
	if err != nil {
		fmt.Printf("Error reading URLs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d URLs from %s\n", len(urls), cfg.InputFile)
	fmt.Println()

	// Initialize components
	fetcher := fetcher.NewFetcher(cfg.Timeout, cfg.MaxRetries, cfg.RateLimit, cfg.Verbose)
	parser := parser.NewParser(cfg.Verbose)
	storage := storage.NewStorage(cfg.OutputDir, cfg.Verbose)
	workerPool := worker.NewPool(cfg.Workers, cfg.RateLimit, cfg.Verbose)

	// Set total for statistics
	storage.SetTotal(len(urls))

	// Start processing
	fmt.Println("Starting concurrent processing...")
	fmt.Println("Press Ctrl+C to stop gracefully")
	fmt.Println()

	startTime := time.Now()

	// Process URLs through worker pool
	results := workerPool.Process(urls, func(url string) (any, error) {
		// Fetch HTML
		fetchResult, err := fetcher.Fetch(url)
		if err != nil {
			return nil, fmt.Errorf("fetch failed: %w", err)
		}

		if fetchResult.Error != nil {
			return nil, fmt.Errorf("HTTP error: %w", fetchResult.Error)
		}

		// Parse HTML
		metadata, err := parser.Parse(fetchResult.Body, url)
		if err != nil {
			return nil, fmt.Errorf("parse failed: %w", err)
		}

		return metadata, nil
	})

	// Process results and save them
	saveErr := make(chan error, 1)
	go func() {
		if err := storage.SaveBatch(results); err != nil {
			saveErr <- err
		} else {
			saveErr <- nil
		}
	}()

	// Wait for all processing to complete
	// First, ensure task generator has sent all tasks
	// Then wait for workers to process them
	// Finally close result channel
	workerPool.Stop()

	// Wait for save operation to complete
	if err := <-saveErr; err != nil {
		fmt.Printf("Error saving batch: %v\n", err)
	}

	// Save final statistics
	if err := storage.SaveStats(); err != nil {
		fmt.Printf("Error saving stats: %v\n", err)
	}

	// Print final statistics
	totalTime := time.Since(startTime)
	fmt.Println()
	fmt.Println("=== Processing Complete ===")
	fmt.Printf("Total time: %v\n", totalTime.Round(time.Second))

	storage.PrintStats()

	fmt.Println()
	fmt.Println("JSON files saved to:", cfg.OutputDir)
}

func readURLs(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" && !strings.HasPrefix(url, "#") {
			urls = append(urls, url)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return urls, nil
}
