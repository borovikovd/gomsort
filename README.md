# gomsort

[![CI](https://github.com/borovikovd/gomsort/actions/workflows/ci.yml/badge.svg)](https://github.com/borovikovd/gomsort/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/borovikovd/gomsort)](https://goreportcard.com/report/github.com/borovikovd/gomsort)
[![codecov](https://codecov.io/gh/borovikovd/gomsort/branch/main/graph/badge.svg)](https://codecov.io/gh/borovikovd/gomsort)

A Go tool that sorts methods within types for better code readability. It reads which methods call which, and puts entry points before the helpers they use.

## Features

- **Method sorting by call graph**: entry points first, the helpers they call after them
- **In place**: each type's methods are reordered among the places they already hold; types, functions and other declarations don't move
- **CLI and analyzer**: a `gofmt`-style command, and a `go/analysis` analyzer for your own driver
- **Safe**: only declarations move, with their comments; generated files, test files, `testdata` and `vendor` are left alone

## Sorting Algorithm

Within each type, methods are sorted by:

1. **Exported First**: Public methods appear before private methods
2. **Call Depth**: Entry points come before the helpers they call, and those before the helpers they call in turn
3. **In-Degree**: Among methods at the same depth, those used by more methods come later
4. **Original Position**: Stable sort fallback

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

**Note**: Like `go fmt`, gomsort processes directories recursively by default. It skips `_test.go` files, generated files (those marked `// Code generated ... DO NOT EDIT.`), and directories the go command ignores: `testdata`, `vendor`, and names starting with `.` or `_`.

### As a check in CI

`-n` prints a line for each file it would change and nothing otherwise, so a check can fail on any output:

```bash
out=$(gomsort -n .) && [ -z "$out" ] || { echo "$out"; exit 1; }
```

### As an analyzer

`github.com/borovikovd/gomsort/pkg/analyzer` provides a `go/analysis` analyzer named `msort` that reports files whose methods would be reordered, for use with `singlechecker`, `multichecker` or a custom golangci-lint build. It isn't one of golangci-lint's built-in linters.

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

## Development

### Prerequisites
- Go 1.24 or later
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
2. **Build Call Graph**: Record which methods each method calls, or passes on as a value, through its receiver (`s.connect()`, `run(s.serve)`). Only calls between methods of the same type, in the same file, count; recursion doesn't.
3. **Calculate Metrics**:
   - **InDegree**: Number of distinct methods that call this method
   - **MaxDepth**: Longest chain of calls from an entry point (a method nothing calls) to this method; methods that call each other share a depth
4. **Sort Methods**: Apply the sorting criteria to each type's methods, and put them back in the places that type's methods held

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by code organization principles from Clean Code and other software engineering best practices
- Built using Go's excellent AST and static analysis packages