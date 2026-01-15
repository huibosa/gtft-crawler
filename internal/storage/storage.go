package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gtft-crawler/internal/parser"
	"gtft-crawler/internal/worker"
)

type Storage struct {
	outputDir string
	fileLock  sync.RWMutex
	stats     *Stats
	verbose   bool
}

type Stats struct {
	Total      int
	Saved      int
	Failed     int
	Skipped    int
	StartTime  time.Time
	LastUpdate time.Time
}

func NewStorage(outputDir string, verbose bool) *Storage {
	return &Storage{
		outputDir: outputDir,
		stats: &Stats{
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
		},
		verbose: verbose,
	}
}

func (s *Storage) Save(metadata *parser.PaperMetadata) error {
	if metadata == nil {
		return fmt.Errorf("metadata is nil")
	}

	// Validate required fields
	if !metadata.Validate() {
		s.stats.Skipped++
		if s.verbose {
			fmt.Printf("Skipping invalid metadata for URL: %s\n", metadata.URL)
		}
		return fmt.Errorf("metadata validation failed")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(s.outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename from article ID
	filename := filepath.Join(s.outputDir, metadata.ID+".json")

	// Acquire lock for this specific file
	s.fileLock.Lock()
	defer s.fileLock.Unlock()

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		if s.verbose {
			fmt.Printf("File already exists, skipping: %s\n", filename)
		}
		s.stats.Skipped++
		return nil
	}

	// Create temporary file for atomic write
	tempFile := filename + ".tmp"

	// Write JSON to temporary file
	if err := s.writeJSON(tempFile, metadata); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		s.stats.Failed++
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	// Atomically rename temp file to final filename
	if err := os.Rename(tempFile, filename); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		s.stats.Failed++
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	s.stats.Saved++
	s.stats.LastUpdate = time.Now()

	if s.verbose {
		fmt.Printf("Saved metadata to: %s\n", filename)
	}

	return nil
}

func (s *Storage) writeJSON(filename string, metadata *parser.PaperMetadata) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func (s *Storage) SaveBatch(results <-chan worker.Result) error {
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Process results concurrently
	for result := range results {
		wg.Add(1)
		go func(r worker.Result) {
			defer wg.Done()

			if r.Error != nil {
				if s.verbose {
					fmt.Printf("Task failed: %s, error: %v\n", r.Task.URL, r.Error)
				}
				s.stats.Failed++
				return
			}

			metadata, ok := r.Data.(*parser.PaperMetadata)
			if !ok {
				err := fmt.Errorf("invalid data type for URL: %s", r.Task.URL)
				errors <- err
				s.stats.Failed++
				return
			}

			if err := s.Save(metadata); err != nil {
				errors <- fmt.Errorf("failed to save metadata for URL %s: %w", r.Task.URL, err)
			}
		}(result)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Collect errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		return fmt.Errorf("batch save completed with %d errors", len(errorList))
	}

	return nil
}

func (s *Storage) SaveStats() error {
	statsFile := filepath.Join(s.outputDir, "stats.json")

	stats := struct {
		Total       int       `json:"total"`
		Saved       int       `json:"saved"`
		Failed      int       `json:"failed"`
		Skipped     int       `json:"skipped"`
		SuccessRate float64   `json:"success_rate"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
		Duration    string    `json:"duration"`
	}{
		Total:       s.stats.Total,
		Saved:       s.stats.Saved,
		Failed:      s.stats.Failed,
		Skipped:     s.stats.Skipped,
		SuccessRate: float64(s.stats.Saved) / float64(s.stats.Total) * 100,
		StartTime:   s.stats.StartTime,
		EndTime:     time.Now(),
		Duration:    time.Since(s.stats.StartTime).String(),
	}

	file, err := os.Create(statsFile)
	if err != nil {
		return fmt.Errorf("failed to create stats file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(stats); err != nil {
		return fmt.Errorf("failed to encode stats JSON: %w", err)
	}

	return nil
}

func (s *Storage) SetTotal(total int) {
	s.stats.Total = total
}

func (s *Storage) GetStats() *Stats {
	return s.stats
}

func (s *Storage) PrintStats() {
	total := s.stats.Saved + s.stats.Failed + s.stats.Skipped
	elapsed := time.Since(s.stats.StartTime)

	fmt.Println("\n=== Storage Statistics ===")
	fmt.Printf("Total processed: %d\n", total)
	fmt.Printf("Successfully saved: %d\n", s.stats.Saved)
	fmt.Printf("Failed: %d\n", s.stats.Failed)
	fmt.Printf("Skipped: %d\n", s.stats.Skipped)

	if total > 0 {
		successRate := float64(s.stats.Saved) / float64(total) * 100
		fmt.Printf("Success rate: %.1f%%\n", successRate)
	}

	fmt.Printf("Elapsed time: %v\n", elapsed.Round(time.Second))

	if s.stats.Saved > 0 {
		avgTime := elapsed / time.Duration(s.stats.Saved)
		fmt.Printf("Average time per save: %v\n", avgTime.Round(time.Millisecond))
	}
}
