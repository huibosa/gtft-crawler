package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type Config struct {
	InputFile  string
	OutputDir  string
	Workers    int
	RateLimit  int
	Timeout    time.Duration
	MaxRetries int
	Verbose    bool
}

func New() *Config {
	return &Config{
		Workers:    20,
		RateLimit:  5,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		OutputDir:  "data/output/all",
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.InputFile, "input", "", "Path to file containing URLs (required)")
	flag.StringVar(&c.OutputDir, "output", c.OutputDir, "Output directory for JSON files")
	flag.IntVar(&c.Workers, "workers", c.Workers, "Number of concurrent workers")
	flag.IntVar(&c.RateLimit, "rate", c.RateLimit, "Maximum requests per second")
	flag.DurationVar(&c.Timeout, "timeout", c.Timeout, "HTTP request timeout")
	flag.IntVar(&c.MaxRetries, "retries", c.MaxRetries, "Maximum retry attempts")
	flag.BoolVar(&c.Verbose, "verbose", false, "Enable verbose logging")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -input data/article_links.txt -workers 30 -rate 10\n", os.Args[0])
	}

	flag.Parse()

	if c.InputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -input flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if c.Workers <= 0 {
		fmt.Fprintf(os.Stderr, "Error: workers must be greater than 0\n")
		os.Exit(1)
	}

	if c.RateLimit <= 0 {
		fmt.Fprintf(os.Stderr, "Error: rate must be greater than 0\n")
		os.Exit(1)
	}

	if c.Timeout <= 0 {
		fmt.Fprintf(os.Stderr, "Error: timeout must be greater than 0\n")
		os.Exit(1)
	}
}
