# gomsort

[![CI](https://github.com/borovikovd/gomsort/actions/workflows/ci.yml/badge.svg)](https://github.com/borovikovd/gomsort/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/borovikovd/gomsort)](https://goreportcard.com/report/github.com/borovikovd/gomsort)
[![codecov](https://codecov.io/gh/borovikovd/gomsort/branch/main/graph/badge.svg)](https://codecov.io/gh/borovikovd/gomsort)

A Go tool that sorts methods within types for better code readability. The tool analyzes call graphs and method usage patterns to optimize method ordering.

## Features

- **Intelligent Method Sorting**: Orders methods based on call depth and usage patterns
- **Call Graph Analysis**: Builds dependency graphs to identify entry points and helpers
- **Multiple Integration Options**: Standalone CLI tool + golangci-lint analyzer
- **Configurable**: Customize sorting criteria via configuration files
- **Safe**: Preserves code semantics while improving readability

## Sorting Algorithm

Methods are sorted by the following criteria:

1. **Receiver Type**: Methods are grouped by their receiver type (alphabetical)
2. **Exported First**: Public methods appear before private methods
3. **Call Depth**: Entry points (low depth) come before deep helpers
4. **In-Degree**: Shared helpers (high in-degree) appear last
5. **Original Position**: Stable sort fallback

This means:
- Public entry points appear at the top
- Deep internal helpers appear near the bottom  
- Shared utility methods appear at the bottom

## Installation

### Using go install (recommended)
```bash
go install github.com/borovikovd/gomsort@latest
```

### Download pre-built binaries
Download from the [releases page](https://github.com/borovikovd/gomsort/releases).

### Build from source
```bash
git clone https://github.com/borovikovd/gomsort.git
cd gomsort
make build
```

## Usage

### Command Line

```bash
# Sort methods in a single file
gomsort file.go

# Sort methods in all Go files in current directory (recursive by default)
gomsort .

# Sort methods in a specific directory tree
gomsort ./src/

# Dry run to see what would be changed
gomsort -n file.go

# Verbose output
gomsort -v file.go
```

### Options

- `-n`: Dry run - show what would be changed without modifying files
- `-v`: Verbose output

**Note**: Like `go fmt`, gomsort processes directories recursively by default.

### Integration with golangci-lint

Add to your `.golangci.yml`:

```yaml
linters:
  enable:
    - msort

linters-settings:
  msort:
    # Configuration options here
```

## Example

**Before:**
```go
type Server struct {
    addr string
}

func (s *Server) helper() string {
    return "help"
}

func (s *Server) Start() error {
    return s.connect()
}

func (s *Server) connect() error {
    s.helper()
    return nil
}

func (s *Server) Stop() error {
    return nil
}
```

**After:**
```go
type Server struct {
    addr string
}

func (s *Server) Start() error {
    return s.connect()
}

func (s *Server) Stop() error {
    return nil
}

func (s *Server) connect() error {
    s.helper()
    return nil
}

func (s *Server) helper() string {
    return "help"
}
```

## Configuration

Create a `.msort.json` file in your project root:

```json
{
  "sort_criteria": {
    "group_by_receiver": true,
    "exported_first": true,
    "sort_by_depth": true,
    "sort_by_in_degree": true,
    "preserve_original_order": true
  },
  "exclude": ["*_test.go"],
  "include": ["*.go"]
}
```

## Development

### Prerequisites
- Go 1.21 or later
- make (optional, for convenience)

### Building
```bash
make build
```

### Testing
```bash
make test
make test-coverage
```

### Linting
```bash
make lint
make lint-fix
```

### Development Workflow
```bash
make dev  # fmt + lint + test
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and linting (`make dev`)
4. Commit your changes (`git commit -am 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## Algorithm Details

The tool performs the following analysis:

1. **Parse AST**: Extract all method declarations and their receivers
2. **Build Call Graph**: Analyze method calls to build dependency relationships
3. **Calculate Metrics**:
   - **InDegree**: Number of distinct methods that call this method
   - **MaxDepth**: Longest call chain where this method appears
4. **Sort Methods**: Apply sorting criteria to optimize readability

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by code organization principles from Clean Code and other software engineering best practices
- Built using Go's excellent AST and static analysis packages