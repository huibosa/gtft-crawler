package worker

import (
	"time"
)

type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskProcessing
	TaskCompleted
	TaskFailed
	TaskSkipped
)

type Task struct {
	ID       string
	URL      string
	Attempts int
	Status   TaskStatus
	Created  time.Time
}

type Result struct {
	Task  Task
	Data  interface{}
	Error error
	Time  time.Duration
}

type Stats struct {
	Total       int
	Completed   int
	Failed      int
	Skipped     int
	SuccessRate float64
	AvgTime     time.Duration
	StartTime   time.Time
	ETA         time.Time
}

func NewTask(id, url string) Task {
	return Task{
		ID:      id,
		URL:     url,
		Status:  TaskPending,
		Created: time.Now(),
	}
}
