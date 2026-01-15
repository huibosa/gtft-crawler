package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ProcessFunc func(url string) (any, error)

type WorkerPool struct {
	workers     int
	rateLimit   int
	taskQueue   chan Task
	resultChan  chan Result
	wg          sync.WaitGroup
	taskGenWg   sync.WaitGroup
	stats       *Stats
	ctx         context.Context
	cancel      context.CancelFunc
	verbose     bool
	rateLimiter *rate.Limiter
}

func NewPool(workers, rateLimit int, verbose bool) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers:     workers,
		rateLimit:   rateLimit,
		taskQueue:   make(chan Task, 1000),
		resultChan:  make(chan Result, 1000),
		stats:       &Stats{StartTime: time.Now()},
		ctx:         ctx,
		cancel:      cancel,
		verbose:     verbose,
		rateLimiter: rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
	}
}

func (wp *WorkerPool) Process(urls []string, processFunc ProcessFunc) <-chan Result {
	wp.stats.Total = len(urls)

	// Start workers
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go func() { wp.worker(processFunc) }()
	}

	// Start task generator
	wp.taskGenWg.Add(1)
	go func() {
		defer wp.taskGenWg.Done()
		wp.generateTasks(urls)
	}()

	// Start stats monitor
	if wp.verbose {
		go wp.monitorStats()
	}

	return wp.resultChan
}

func (wp *WorkerPool) generateTasks(urls []string) {
	if wp.verbose {
		fmt.Printf("Task generator started, processing %d URLs\n", len(urls))
	}

	sent := 0
	for _, url := range urls {
		task := NewTask(extractIDFromURL(url), url)
		select {
		case wp.taskQueue <- task:
			sent++
			if wp.verbose && sent%100 == 0 {
				fmt.Printf("Task generator: sent %d/%d tasks\n", sent, len(urls))
			}
		case <-wp.ctx.Done():
			if wp.verbose {
				fmt.Printf("Task generator: context cancelled, sent %d/%d tasks\n", sent, len(urls))
			}
			return
		}
	}
	close(wp.taskQueue)

	if wp.verbose {
		fmt.Printf("Task generator: completed, sent all %d tasks\n", len(urls))
	}
}

func (wp *WorkerPool) worker(processFunc ProcessFunc) {
	defer wp.wg.Done()

	if wp.verbose {
		fmt.Printf("Worker started\n")
	}

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				if wp.verbose {
					fmt.Printf("Worker: task queue closed, exiting\n")
				}
				return
			}

			if wp.verbose {
				fmt.Printf("Worker: processing task %s\n", task.ID)
			}

			// Apply shared rate limiting
			if err := wp.rateLimiter.Wait(wp.ctx); err != nil {
				if wp.verbose {
					fmt.Printf("Worker: context cancelled, exiting\n")
				}
				return
			}

			start := time.Now()
			task.Status = TaskProcessing

			// Handle panics in processFunc
			data, err := func() (data any, err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic in processFunc: %v", r)
					}
				}()
				return processFunc(task.URL)
			}()

			duration := time.Since(start)

			result := Result{
				Task:  task,
				Data:  data,
				Error: err,
				Time:  duration,
			}

			wp.updateStats(result)

			select {
			case wp.resultChan <- result:
				if wp.verbose && result.Error != nil {
					fmt.Printf("Worker: task %s failed: %v\n", task.ID, result.Error)
				}
			case <-wp.ctx.Done():
				if wp.verbose {
					fmt.Printf("Worker: context cancelled while sending result, exiting\n")
				}
				return
			}

		case <-wp.ctx.Done():
			if wp.verbose {
				fmt.Printf("Worker: context cancelled, exiting\n")
			}
			return
		}
	}
}

func (wp *WorkerPool) processTask(task Task, processFunc ProcessFunc) {
	// Apply shared rate limiting (non-blocking)
	ctx, cancel := context.WithTimeout(wp.ctx, 100*time.Millisecond)
	defer cancel()
	wp.rateLimiter.Wait(ctx)

	start := time.Now()
	task.Status = TaskProcessing

	// Handle panics in processFunc
	data, err := func() (data any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in processFunc: %v", r)
			}
		}()
		return processFunc(task.URL)
	}()

	duration := time.Since(start)

	result := Result{
		Task:  task,
		Data:  data,
		Error: err,
		Time:  duration,
	}

	wp.updateStats(result)

	// Try to send result non-blockingly
	select {
	case wp.resultChan <- result:
	case <-time.After(100 * time.Millisecond):
		// Drop result if channel is full
	}
}

func (wp *WorkerPool) updateStats(result Result) {
	wp.stats.AvgTime = (wp.stats.AvgTime*time.Duration(wp.stats.Completed+wp.stats.Failed) + result.Time) / time.Duration(wp.stats.Completed+wp.stats.Failed+1)

	if result.Error != nil {
		wp.stats.Failed++
		result.Task.Status = TaskFailed
	} else {
		wp.stats.Completed++
		result.Task.Status = TaskCompleted
	}

	completed := wp.stats.Completed + wp.stats.Failed
	if completed > 0 {
		wp.stats.SuccessRate = float64(wp.stats.Completed) / float64(completed) * 100

		elapsed := time.Since(wp.stats.StartTime)
		avgTimePerTask := elapsed / time.Duration(completed)
		remainingTasks := wp.stats.Total - completed
		eta := time.Now().Add(avgTimePerTask * time.Duration(remainingTasks))
		wp.stats.ETA = eta
	}
}

func (wp *WorkerPool) monitorStats() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wp.printStats()
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) printStats() {
	completed := wp.stats.Completed + wp.stats.Failed
	progress := float64(completed) / float64(wp.stats.Total) * 100

	fmt.Println("\n====================================================================")
	fmt.Printf("\rProgress: %d/%d (%.1f%%) | Success: %.1f%% | Avg: %v | ETA: %v\n",
		completed, wp.stats.Total, progress, wp.stats.SuccessRate,
		wp.stats.AvgTime.Round(time.Millisecond), wp.stats.ETA.Format("15:04:05"))
	fmt.Println("====================================================================")
}

func (wp *WorkerPool) Stop() {
	// Wait for task generator to finish sending all tasks
	wp.taskGenWg.Wait()

	// Task generator already closed wp.taskQueue
	// Now wait for workers to finish processing all tasks
	wp.wg.Wait()

	// Close the result channel after all workers are done
	close(wp.resultChan)

	if wp.verbose {
		fmt.Println()
		wp.printFinalStats()
	}
}

func (wp *WorkerPool) printFinalStats() {
	totalTime := time.Since(wp.stats.StartTime)

	fmt.Println("\n=== Processing Complete ===")
	fmt.Printf("Total URLs:      %d\n", wp.stats.Total)
	fmt.Printf("Completed:       %d (%.1f%%)\n", wp.stats.Completed, float64(wp.stats.Completed)/float64(wp.stats.Total)*100)
	fmt.Printf("Failed:          %d (%.1f%%)\n", wp.stats.Failed, float64(wp.stats.Failed)/float64(wp.stats.Total)*100)
	fmt.Printf("Success Rate:    %.1f%%\n", wp.stats.SuccessRate)
	fmt.Printf("Average Time:    %v\n", wp.stats.AvgTime.Round(time.Millisecond))
	fmt.Printf("Total Time:      %v\n", totalTime.Round(time.Second))
	fmt.Printf("Requests/sec:    %.1f\n", float64(wp.stats.Total)/totalTime.Seconds())
}

func extractIDFromURL(url string) string {
	// Extract ID from URL patterns:
	// 1. https://www.gtft.cn/article/id/{uuid}
	// 2. https://www.gtft.cn/cn/article/id/{uuid}
	// 3. https://www.gtft.cn/cn/article/doi/{doi}

	// Try to extract UUID from /article/id/ pattern
	if idx := strings.LastIndex(url, "/article/id/"); idx != -1 {
		return url[idx+len("/article/id/"):]
	}

	// Try to extract DOI from /article/doi/ pattern
	if idx := strings.LastIndex(url, "/article/doi/"); idx != -1 {
		return url[idx+len("/article/doi/"):]
	}

	// Fallback: return the URL itself
	return url
}
