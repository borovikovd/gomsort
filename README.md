# gomsort

[![CI](https://github.com/borovikovd/gomsort/actions/workflows/ci.yml/badge.svg)](https://github.com/borovikovd/gomsort/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/borovikovd/gomsort)](https://goreportcard.com/report/github.com/borovikovd/gomsort)
[![codecov](https://codecov.io/gh/borovikovd/gomsort/branch/main/graph/badge.svg)](https://codecov.io/gh/borovikovd/gomsort)

A Go tool that sorts methods within types the way Go code usually reads: each type's methods together, exported ones first, then the helpers in the order they're used.

## Features

- **Method sorting by call graph**: exported methods first, then the helpers in call order
- **Grouped by type**: each type's methods gather where its first method is; helper functions that sat between them follow the block, and nothing before it moves
- **CLI and analyzer**: a `gofmt`-style command, and a `go/analysis` analyzer for your own driver
- **Safe**: only declarations move, with their comments; generated files, test files, `testdata` and `vendor` are left alone

## Sorting Algorithm

Within each type:

1. **Exported first**: exported methods, in their current order.
2. **Then call order**: the unexported methods the exported ones use, in the order they first use them, each followed by its own helpers, depth first. Then the other unexported entry points, used from outside the type (by a function or another type's methods in the file) or not at all, in their current order, each followed by its helpers.
3. **Shared helpers once**: a helper several methods use follows the first of them.

Each type's methods form one block where its first method is. Declarations that sat between them, such as helper functions, follow the block in their order; nothing before a type's first method moves.

This means:
- A type's public methods come first and its helpers after them, top-down
- Plain helper functions end up after the methods that use them
- An edit moves only the methods whose first user changes; a method gaining another user stays put

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
2. **Find Uses**: Record which methods each function and method uses, by calling them or passing them on as values (`g.add(c)`, `run(s.serve)`), in the order it first uses them. A use counts when the file shows the value's type: the receiver, parameters and results, `var` declarations, `T{...}`, `&T{...}`, `new(T)`, and ranges over slices or maps of `T`. Uses through struct fields or from other files aren't seen; recursion doesn't count.
3. **Order**: For each type, place its exported methods, then its unexported methods in call order, depth first
4. **Rewrite**: Gather each type's methods where its first method is, with whatever sat between them after the block

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by code organization principles from Clean Code and other software engineering best practices
- Built using Go's excellent AST and static analysis packages