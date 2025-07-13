# API Spray

A high-performance API endpoint discovery tool built in Go with intelligent false positive detection and resume capabilities.

## Features

- **Multiple Scan Modes**: Wildcards, directories, and subdomains
- **High Performance**: Concurrent processing with configurable threads
- **False Positive Detection**: Automatically filters out common false positives
- **Resume Functionality**: Continue interrupted scans from where they left off
- **Flexible Output**: CSV results with detailed response information
- **HTTP/HTTPS Support**: Automatic protocol detection and fallback
- **Customizable**: Configurable timeouts, retries, and status codes

## Installation

### From Source

```bash
git clone https://github.com/davidwkirsch/api_spray.git
cd api_spray
go install
```

This will install `api_spray` to your `$GOPATH/bin` directory, making it available as a global command.

### Direct Install

If you have Go installed, you can install directly:

```bash
go install github.com/davidwkirsch/api_spray@latest
```

### Prerequisites

- Go 1.19 or later
- `$GOPATH/bin` in your `$PATH` environment variable

## Quick Start

Basic usage with default settings:

```bash
api_spray -targets targets.txt -wordlist words.txt
```

High-performance scan with custom settings:

```bash
api_spray -targets targets.txt -wordlist words.txt -threads 100 -batch 20 -timeout 5s
```

## Command Line Options

### Required Arguments

| Flag | Description | Example |
|------|-------------|---------|
| `-targets` | File containing target domains | `-targets domains.txt` |
| `-wordlist` | Wordlist file for endpoint discovery | `-wordlist api_endpoints.txt` |

### Scan Configuration

| Flag | Default | Description | Example |
|------|---------|-------------|---------|
| `-mode` | `wildcards` | Scan mode: `wildcards`, `directories`, `subdomains` | `-mode directories` |
| `-threads` | `50` | Number of concurrent threads | `-threads 100` |
| `-batch` | `10` | Number of words processed per batch | `-batch 20` |
| `-timeout` | `10s` | HTTP request timeout | `-timeout 5s` |

### HTTP Configuration

| Flag | Default | Description | Example |
|------|---------|-------------|---------|
| `-disable-http` | `false` | Disable HTTP fallback (HTTPS only) | `-disable-http` |
| `-follow-redirects` | `true` | Follow HTTP redirects | `-follow-redirects=false` |
| `-retries` | `1` | Maximum retries per request | `-retries 3` |
| `-user-agent` | `Mozilla/5.0 (compatible; api_spray/1.0)` | Custom user agent | `-user-agent "MyBot/1.0"` |
| `-status-codes` | `200` | Success status codes (comma-separated) | `-status-codes "200,201,204"` |

### Output and Resume

| Flag | Default | Description | Example |
|------|---------|-------------|---------|
| `-outdir` | `results` | Output directory for results | `-outdir /tmp/scan_results` |
| `-resume` | `false` | Resume previous scan | `-resume` |

## Scan Modes

### Wildcards Mode (Default)

Scans for API endpoints using wildcard patterns:

```bash
api_spray -targets targets.txt -wordlist words.txt -mode wildcards
```

**Example URLs generated:**
- `https://example.com/api/users`
- `https://example.com/v1/admin`
- `https://example.com/api/v2/health`

### Directories Mode

Scans for directory-based endpoints:

```bash
api_spray -targets targets.txt -wordlist words.txt -mode directories
```

**Example URLs generated:**
- `https://example.com/admin/`
- `https://example.com/api/`
- `https://example.com/v1/`

### Subdomains Mode

Scans for subdomain-based endpoints:

```bash
api_spray -targets targets.txt -wordlist words.txt -mode subdomains
```

**Example URLs generated:**
- `https://api.example.com/`
- `https://admin.example.com/`
- `https://v1.example.com/`

## Input Files

### Targets File

Create a file with one domain per line:

```
example.com
api.company.com
test.domain.org
```

### Wordlist File

Create a file with one word/endpoint per line:

```
api
admin
v1
v2
users
health
status
login
```

## Examples

### Basic API Discovery

```bash
api_spray -targets domains.txt -wordlist api_words.txt
```

### High-Performance Scan

```bash
api_spray -targets domains.txt -wordlist large_wordlist.txt -threads 200 -batch 50 -timeout 3s
```

### Resume Interrupted Scan

```bash
api_spray -targets domains.txt -wordlist words.txt -resume
```

### Custom Status Codes

```bash
api_spray -targets domains.txt -wordlist words.txt -status-codes "200,201,204,301,302"
```

### HTTPS Only Scan

```bash
api_spray -targets domains.txt -wordlist words.txt -disable-http
```

### Subdomain Discovery

```bash
api_spray -targets domains.txt -wordlist subdomains.txt -mode subdomains -threads 100
```

## Output

### CSV Results

Results are saved in CSV format with the following columns:

- `target`: Target domain
- `word`: Word from wordlist
- `url`: Full URL tested
- `status_code`: HTTP status code
- `content_length`: Response size in bytes
- `response_time_ms`: Response time in milliseconds
- `title`: HTML title (if available)
- `error`: Error message (if any)

### Directory Structure

```
results/
├── results.csv          # Main results file
├── progress.json        # Progress tracking for resume
├── errors.log          # Error log
└── scan.log            # Detailed scan log
```

## False Positive Detection

API Spray automatically detects and filters false positives by:

1. **Response Size Tracking**: Monitors response sizes for each target and status code
2. **Threshold-Based Filtering**: Filters responses that appear more than 10 times with the same size
3. **Adaptive Learning**: Continuously learns patterns during the scan

## Performance Tips

1. **Adjust Thread Count**: Start with 50 threads and increase based on your system and network
2. **Optimize Batch Size**: Larger batches (20-50) can improve performance for large wordlists
3. **Set Appropriate Timeout**: Use shorter timeouts (3-5s) for faster scans
4. **Use Resume Feature**: For large scans, use `-resume` to continue interrupted scans

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This tool is intended for authorized security testing only. Users are responsible for ensuring they have proper authorization before scanning any targets.
