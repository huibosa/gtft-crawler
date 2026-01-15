# GTFT Academic Paper Crawler

A high-performance, concurrent web crawler for extracting metadata from GTFT (钢铁钒钛) journal articles. This tool automates the collection of academic paper metadata for research analysis, bibliometric studies, and academic database population.

## Features

- **Concurrent Processing**: Configurable worker pool for parallel URL processing
- **Intelligent Rate Limiting**: Prevents server overload with adjustable request limits
- **Automatic Retry Logic**: Exponential backoff with configurable retry attempts
- **Dual URL Format Support**: Handles both UUID-based and DOI-based article URLs
- **Structured JSON Output**: Comprehensive metadata in standardized JSON format
- **Progress Tracking**: Real-time statistics and progress monitoring
- **Robust Error Handling**: Graceful degradation with detailed error reporting

## Installation

### Prerequisites
- Go 1.25.5 or higher

### From Source
```bash
# Clone the repository
git clone <repository-url>
cd gtft-crawler

# Build the binary
go build -o gtft-crawler main.go

# Or install globally
go install .
```

### Cross-Compilation for Fedora x86-64
```bash
# Create a statically linked binary for Fedora
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-s -w" -o gtft-crawler main.go

# Verify the binary
file gtft-crawler
# Should output: ELF 64-bit LSB executable, x86-64, statically linked, stripped
```

## Usage

### Basic Command
```bash
./gtft-crawler -input data/article_links.txt -workers 20 -rate 5
```

### Command-Line Options
| Option | Description | Default |
|--------|-------------|---------|
| `-input` | Path to file containing URLs **(required)** | - |
| `-output` | Output directory for JSON files | `data/output/all` |
| `-workers` | Number of concurrent workers | `20` |
| `-rate` | Maximum requests per second | `5` |
| `-timeout` | HTTP request timeout | `30s` |
| `-retries` | Maximum retry attempts | `3` |
| `-verbose` | Enable verbose logging | `false` |

### Example
```bash
./gtft-crawler \
  -input data/article_links.txt \
  -workers 30 \
  -rate 10 \
  -timeout 60s \
  -retries 5 \
  -verbose
```

## Input Format

Create a text file with one URL per line. The crawler supports two URL formats:

### UUID Format (Legacy)
```
https://www.gtft.cn/article/id/fc9d8b76-87b6-494f-9de1-5d968b3b54cd
https://www.gtft.cn/cn/article/id/004bb399-06fc-4db0-9ef3-b2b2438967b3
```

### DOI Format (Modern)
```
https://www.gtft.cn/cn/article/doi/10.7513/j.issn.1004-7638.2025.04.001
https://www.gtft.cn/cn/article/doi/10.7513/j.issn.1004-7638.2025.04.025
```

### Example Input File
See [`data/test-links.txt`](data/test-links.txt) for a working example.

## Output Format

Each article is saved as a separate JSON file named `{article_id}.json`. The JSON structure includes:

```json
{
  "id": "fc9d8b76-87b6-494f-9de1-5d968b3b54cd",
  "url": "https://www.gtft.cn/cn/article/id/fc9d8b76-87b6-494f-9de1-5d968b3b54cd",
  "language": "zh",
  "title_cn": "超细晶粒钢力学性能研究",
  "authors": [
    {
      "name": "宋立秋",
      "affiliation": "钢铁研究总院",
      "order": 1
    }
  ],
  "journal_cn": "钢铁钒钛",
  "journal_en": "Iron Steel Vanadium Titanium",
  "journal_abbr": "gtft",
  "issn": "1004-7638",
  "volume": "24",
  "issue": "4",
  "pages": "1-5",
  "year": "2003",
  "date": "2003-12-31",
  "online_date": "2003-09-03",
  "submit_date": "2003-09-03",
  "abstract_cn": "在攀钢1450热连轧机上，生产出了Q235普碳钢成分的超细晶粒热轧钢板...",
  "abstract_en": "Ultra-fine grain hot rolled sheets were produced in 1450 hot mill at PZH Steel...",
  "keywords_cn": ["超细晶粒钢", "组织", "热轧", "力学性能"],
  "keywords_en": ["ultra-fine grain steel", "microstructure", "hot rolling", "mechanical property"],
  "pdf_url": "https://www.gtft.cn/cn/article/id/fc9d8b76-87b6-494f-9de1-5d968b3b54cd",
  "pdf_size": "1.2MB",
  "views": 1250,
  "downloads": 843,
  "citations": 42,
  "doi": "10.7513/j.issn.1004-7638.2003.04.001",
  "fund_project": "国家自然科学基金项目(50274020)",
  "clc_code": "TG142.1",
  "license": "http://creativecommons.org/licenses/by/3.0/",
  "parsed_at": "2025-01-16T10:30:45Z"
}
```

## Project Structure

```
gtft-crawler/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── README.md               # This file
├── internal/               # Core application modules
│   ├── config/            # Configuration management
│   ├── fetcher/           # HTTP fetching with retry logic
│   ├── parser/            # HTML parsing and metadata extraction
│   ├── storage/           # JSON file storage and management
│   └── worker/            # Concurrent worker pool implementation
└── data/                  # Data directories
    ├── article_links.txt  # Example URL list (4226+ URLs)
    ├── test-links.txt     # Small test URL list
    ├── htmls/             # Example HTML files for testing
    └── output/            # Generated JSON files
```

## Use Cases

### 1. **Academic Research**
- **Bibliometric Analysis**: Collect metadata for citation analysis, co-authorship networks, and research trend identification
- **Literature Reviews**: Automate collection of relevant papers for systematic reviews
- **Research Impact Assessment**: Track citations and download statistics over time

### 2. **Institutional Repositories**
- **Library Cataloging**: Populate institutional repositories with standardized metadata
- **Digital Archives**: Batch import academic papers into digital preservation systems
- **Knowledge Management**: Create searchable databases of academic publications

### 3. **Data Science & Analytics**
- **Text Mining**: Prepare structured data for natural language processing (NLP) tasks
- **Topic Modeling**: Analyze abstracts and keywords for thematic clustering
- **Temporal Analysis**: Study publication trends and seasonal patterns



### 4. **Academic Publishing**
- **Journal Management**: Monitor article performance metrics (views, downloads, citations)
- **Editorial Workflow**: Automate metadata extraction for editorial systems
- **CrossRef Integration**: Prepare data for DOI registration and cross-publisher linking



### 5. **Educational Applications**
- **Course Material Compilation**: Gather relevant readings for academic courses
- **Research Training**: Teach students about metadata standards and data collection
- **Academic Writing**: Provide structured references for thesis and dissertation writing



## Performance Considerations

### Default Configuration
- **Workers**: 20 concurrent workers
- **Rate Limit**: 5 requests per second
- **Timeout**: 30 seconds per request
- **Retries**: 3 attempts with exponential backoff



### Expected Throughput
- **Average Processing Time**: 2-3 seconds per URL
- **Concurrent Capacity**: 20 simultaneous HTTP requests
- **Rate-Limited Throughput**: 5 successful requests per second maximum



## Troubleshooting

### Common Issues

#### Network Timeouts
```bash
# Increase timeout duration
./gtft-crawler -input urls.txt -timeout 60s



# Reduce concurrent workers
./gtft-crawler -input urls.txt -workers 10
```

#### Rate Limiting
```bash
# Reduce request rate
./gtf-crawler -input urls.txt -rate 2



# Add delays between batches
# (Consider server's acceptable usage policy)
```

#### Missing Metadata
- **Cause**: HTML structure changes on target website
- **Solution**: Update parser logic in `internal/parser/parser.go`
- **Verification**: Check `data/htmls/` for example HTML files



#### File System Issues
```bash
# Ensure output directory exists
mkdir -p data/output/all



# Check write permissions
ls -la data/output/
```

### Error Messages

| Error | Likely Cause | Solution |
|-------|--------------|----------|
| `fetch failed: context deadline exceeded` | Network timeout | Increase `-timeout` value |
| `HTTP error: 429 Too Many Requests` | Rate limiting | Reduce `-rate` or increase delays |
| `HTTP error: 404 Not Found` | Invalid URL | Verify URL format and availability |
| `failed to create output directory` | Permission issues | Check directory permissions |
| `invalid data type for URL` | Parser mismatch | Update parser for current HTML structure |



## Development

### Building from Source
```bash
# Clone the repository
git clone <repository-url>
cd gtft-crawler



# Install dependencies
go mod download



# Build the application
go build -o gtft-crawler main.go



# Run tests (if available)
go test ./...
```

### Code Structure

#### Main Components

1. **Config (`internal/config/`)**
   - Command-line flag parsing
   - Configuration validation
   - Default value management



2. **Fetcher (`internal/fetcher/`)**
   - HTTP request execution with retry logic
   - Rate limiting implementation
   - Error handling and recovery



3. **Parser (`internal/parser/`)**
   - HTML parsing with goquery
   - Metadata extraction logic
   - Data validation and normalization



4. **Storage (`internal/storage/`)**
   - JSON file writing with atomic operations
   - File existence checking
   - Statistics collection



5. **Worker (`internal/worker/`)**
   - Concurrent task processing
   - Rate limiting coordination
   - Progress monitoring



### Extending the Crawler

#### Adding New Metadata Fields
1. Update `internal/parser/types.go` to add new struct fields
2. Modify `internal/parser/parser.go` to extract new data
3. Update validation logic in `Validate()` method



#### Supporting New URL Formats
1. Update `extractIDFromURL()` in `internal/worker/pool.go`
2. Add new parsing patterns for different URL structures
3. Test with sample URLs



#### Customizing Output Format
1. Modify JSON encoding in `internal/storage/storage.go`
2. Adjust field names and structure in `writeJSON()` method
3. Update any dependent parsing logic



## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.



## Acknowledgments

- Built with Go and the excellent `goquery` library for HTML parsing
- Designed for academic research and data collection purposes
- Inspired by the need for automated metadata extraction in bibliometric studies



## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/improvement`)
3. Commit changes (`git commit -am 'Add new feature'`)
4. Push to branch (`git push origin feature/improvement`)
5. Create a Pull Request



---

**Note**: This tool is designed for responsible web crawling. Please respect the target website's terms of service, robots.txt directives, and rate limits to ensure sustainable operation.