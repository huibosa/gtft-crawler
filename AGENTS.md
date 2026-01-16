# AGENTS.md - GTFT Academic Paper Crawler

This document provides guidelines for AI agents working on the GTFT Academic Paper Crawler project. It includes build commands, code style guidelines, and project-specific conventions.

## Project Overview

GTFT Academic Paper Crawler is a high-performance, concurrent web crawler written in Go for extracting metadata from GTFT (钢铁钒钛) journal articles. The tool automates collection of academic paper metadata for research analysis, bibliometric studies, and academic database population.

## Build & Development Commands

### Building the Application
```bash
# Build the binary
go build -o gtft-crawler main.go

# Install globally
go install .

# Cross-compilation for Fedora x86-64 (statically linked)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-s -w" -o gtft-crawler main.go
```

### Dependency Management
```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test ./internal/parser
```

### Code Quality & Linting
```bash
# Format code
go fmt ./...

# Vet code for suspicious constructs
go vet ./...

# Check for unused dependencies
go mod tidy -v
```

### Running the Application
```bash
# Basic usage
./gtft-crawler -input data/article_links.txt -workers 20 -rate 5

# With custom output directory
./gtft-crawler -input data/article_links.txt -output data/custom_output -workers 30

# Verbose mode with increased timeout
./gtft-crawler -input data/article_links.txt -workers 20 -rate 5 -timeout 60s -verbose
```

## Code Style Guidelines

### Package Structure
- Use `internal/` for application-specific packages that shouldn't be imported by external code
- Follow standard Go project layout conventions
- Keep package names short, lowercase, and descriptive

### Imports Ordering
```go
import (
    // Standard library
    "fmt"
    "os"
    "time"

    // Third-party packages
    "github.com/PuerkitoBio/goquery"
    "golang.org/x/time/rate"

    // Internal packages (project-specific)
    "gtft-crawler/internal/config"
    "gtft-crawler/internal/parser"
)
```

### Naming Conventions
- **Variables**: Use camelCase (e.g., `articleMetadata`, `workerPool`)
- **Constants**: Use CamelCase or UPPER_SNAKE_CASE for exported constants
- **Types**: Use PascalCase (e.g., `PaperMetadata`, `WorkerPool`)
- **Interfaces**: Use PascalCase ending with "er" when appropriate (e.g., `Fetcher`, `Parser`)
- **Methods**: Use PascalCase for exported methods, camelCase for unexported
- **File names**: Use lowercase with underscores for multiple words (e.g., `paper_metadata.go`)

### Error Handling
- Always handle errors explicitly; never ignore them
- Use `fmt.Errorf` with `%w` for wrapping errors: `fmt.Errorf("fetch failed: %w", err)`
- Return early on errors rather than nesting conditionals
- Include context in error messages (what operation failed)
- For HTTP errors, include status code: `fmt.Errorf("HTTP error: %w", fetchResult.Error)`

### Types and Structs
```go
// Use JSON tags for serialization
type Author struct {
    Name        string `json:"name"`
    Affiliation string `json:"affiliation,omitempty"`
    Order       int    `json:"order,omitempty"`
}

// Include field comments for complex structs
type PaperMetadata struct {
    // Core Identification
    ID       string `json:"id"`
    URL      string `json:"url"`
    Language string `json:"language"`
    
    // Titles
    TitleCN string `json:"title_cn"`
    TitleEN string `json:"title_en,omitempty"`
}
```

### Function Design
- Keep functions focused and single-purpose
- Use descriptive names that indicate what the function does
- For constructors, use `New` prefix: `NewParser()`, `NewFetcher()`
- Include parameter validation in public functions
- Document exported functions with comments

### Concurrency Patterns
- Use `sync.WaitGroup` for coordinating goroutines
- Implement rate limiting with `golang.org/x/time/rate`
- Use buffered channels for task queues
- Include graceful shutdown mechanisms
- Protect shared resources with `sync.RWMutex` or `sync.Mutex`

### Logging and Verbosity
- Use the `verbose` flag to control detailed logging
- Print progress updates to stdout for user feedback
- Include timestamps in verbose output when helpful
- Structure logs: `[Component] Message: details`

### HTTP Client Configuration
- Set reasonable timeouts (default: 30 seconds)
- Implement retry logic with exponential backoff
- Include User-Agent header in requests
- Respect rate limits to avoid server overload

## Project-Specific Patterns

### Metadata Extraction
- Parse HTML using `goquery` library
- Extract data from specific CSS selectors
- Validate extracted metadata before saving
- Handle both UUID and DOI URL formats

### Worker Pool Implementation
```go
// Task processing pattern
results := workerPool.Process(urls, func(url string) (any, error) {
    // Fetch HTML
    fetchResult, err := fetcher.Fetch(url)
    if err != nil {
        return nil, fmt.Errorf("fetch failed: %w", err)
    }
    
    // Parse HTML
    metadata, err := parser.Parse(fetchResult.Body, url)
    if err != nil {
        return nil, fmt.Errorf("parse failed: %w", err)
    }
    
    return metadata, nil
})
```

### File Storage
- Save each article as separate JSON file
- Use atomic file operations to prevent corruption
- Include statistics collection
- Validate JSON structure before writing

### Configuration Management
- Use `flag` package for command-line arguments
- Set sensible defaults for all parameters
- Validate required flags (e.g., `-input` is required)
- Include usage information with examples

## Testing Guidelines

### Unit Tests
- Place tests in same package as code being tested
- Use `_test.go` suffix for test files
- Test both success and error cases
- Mock external dependencies when appropriate

### Integration Tests
- Test end-to-end workflow with sample data
- Use `data/test-links.txt` for integration testing
- Verify JSON output structure matches expected format
- Test concurrent processing with various worker counts

### Test Data
- Sample HTML files in `data/htmls/` directory
- Test URL list in `data/test-links.txt`
- Expected output patterns in README examples

## Performance Considerations

### Default Configuration
- Workers: 20 concurrent workers
- Rate Limit: 5 requests per second
- Timeout: 30 seconds per request
- Retries: 3 attempts with exponential backoff

### Optimization Guidelines
- Profile before optimizing
- Use connection pooling for HTTP client
- Implement caching for repeated operations
- Monitor memory usage with large datasets

## Security Best Practices

### Input Validation
- Validate URLs before processing
- Sanitize HTML input to prevent injection
- Limit file system access to designated directories
- Validate JSON structure before parsing

### Network Security
- Use HTTPS for all external requests
- Implement timeout and cancellation contexts
- Limit concurrent connections to prevent DoS
- Respect robots.txt and terms of service

### File System Security
- Restrict file permissions to necessary operations
- Validate file paths to prevent directory traversal
- Use atomic operations for file writes
- Clean up temporary files

## Development Workflow

### Adding New Features
1. Update `internal/parser/types.go` to add new struct fields
2. Modify `internal/parser/parser.go` to extract new data
3. Update validation logic in `Validate()` method
4. Test with sample HTML files

### Supporting New URL Formats
1. Update `extractIDFromURL()` in `internal/worker/pool.go`
2. Add new parsing patterns for different URL structures
3. Test with sample URLs in both formats

### Code Review Checklist
- [ ] Follows established naming conventions
- [ ] Includes proper error handling
- [ ] Has appropriate test coverage
- [ ] Respects rate limiting and timeout settings
- [ ] Maintains backward compatibility
- [ ] Includes documentation updates if needed

## Troubleshooting

### Common Issues
- Network timeouts: Increase `-timeout` value or reduce workers
- Rate limiting: Reduce `-rate` parameter
- Missing metadata: Update parser logic for current HTML structure
- File system issues: Check directory permissions

### Debugging Tips
- Use `-verbose` flag for detailed logging
- Check `data/htmls/` for example HTML files
- Verify URL format matches supported patterns
- Monitor memory usage with large URL lists

## References

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- Project README: Detailed usage instructions and examples