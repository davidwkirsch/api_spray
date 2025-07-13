# API Spray - Restructured

A high-performance API endpoint discovery tool built in Go, now with a clean, modular architecture.

## Project Structure

```
api-spray/
├── main.go                     # Simple main entry point
├── go.mod                      # Go module definition
├── pkg/                        # Public packages
│   └── types/                  # Shared types and interfaces
│       └── types.go
├── internal/                   # Private application packages
│   ├── config/                 # Configuration handling
│   │   └── config.go
│   ├── http/                   # HTTP client functionality
│   │   └── client.go
│   ├── output/                 # Output management (CSV, logs)
│   │   └── manager.go
│   ├── progress/               # Progress tracking and resume
│   │   └── manager.go
│   └── scanner/                # Main scanning engine
│       └── scanner.go
└── old_main.go                 # Original monolithic file (backup)
```

## Architecture Benefits

### 1. **Separation of Concerns**
- Each package has a single responsibility
- Easy to test individual components
- Clear interfaces between modules

### 2. **Modular Design**
- `pkg/types`: Shared data structures
- `internal/config`: Configuration parsing and file loading
- `internal/http`: HTTP client with retry logic
- `internal/output`: CSV and log file management
- `internal/progress`: Progress tracking and false positive detection
- `internal/scanner`: Main orchestration logic

### 3. **Maintainability**
- Small, focused files instead of one large file
- Easy to add new features without affecting existing code
- Clear package boundaries

### 4. **Extensibility**
- Easy to add new scan modes
- Simple to implement new output formats
- Straightforward to add new HTTP features

## Usage

The tool maintains the same command-line interface:

```bash
go run main.go -targets targets.txt -wordlist words.txt -threads 100
```

## Key Components

### Config Package
- Parses command line flags
- Loads target and wordlist files
- Validates configuration

### HTTP Package
- Manages HTTP client with connection pooling
- Handles retries and timeouts
- Generates URLs based on scan mode
- Extracts titles from HTML responses

### Scanner Package
- Orchestrates the scanning process
- Manages worker goroutines
- Tracks statistics
- Handles batch processing

### Progress Package
- Tracks scan progress for resume functionality
- Implements false positive detection
- Manages completed work tracking

### Output Package
- Writes results to CSV files
- Manages log files
- Handles error filtering

## Future Enhancements

With this structure, you can easily add:

1. **New Scan Modes**: Add to `types.go` and implement in `http/client.go`
2. **Output Formats**: Create new writers in `output/` package
3. **Authentication**: Extend `http/client.go` 
4. **Rate Limiting**: Add to `scanner/scanner.go`
5. **Plugins**: Create new packages in `internal/`
6. **API Server**: Add `cmd/server/` for web interface
7. **Different Backends**: Add `internal/storage/` for databases

## Building

```bash
go build -o api-spray main.go
```

The modular structure makes the codebase much more maintainable and extensible while preserving all existing functionality.
