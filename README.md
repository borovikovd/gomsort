# gomsort

[![CI](https://github.com/borovikovd/gomsort/actions/workflows/ci.yml/badge.svg)](https://github.com/borovikovd/gomsort/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/borovikovd/gomsort)](https://goreportcard.com/report/github.com/borovikovd/gomsort)
[![codecov](https://codecov.io/gh/borovikovd/gomsort/branch/main/graph/badge.svg)](https://codecov.io/gh/borovikovd/gomsort)

A Go tool that sorts methods the way Go code usually reads: exported ones first, then the helpers in the order they're used. It sorts within each run of a type's methods and never moves anything past another declaration, so files laid out in feature sections, each with its types next to the methods that use them, keep their sections.

## Features

- **Method sorting by call graph**: exported methods first, then the helpers in call order
- **Keeps your layout**: only methods move, within each run of one type's methods; types, functions, constants and variables stay where they are
- **CLI and analyzer**: a `gofmt`-style command, and a `go/analysis` analyzer for your own driver
- **Safe**: only whole declarations move, their text copied as it is with the comments around them, then gofmt'd; generated files, test files, `testdata` and `vendor` are left alone

## Sorting Algorithm

A run is a type's methods one after another, with no other declaration, and no free-standing comment such as `// --- Snapshots`, between them. A method's doc comment, or a comment after the previous method's closing brace, doesn't separate a run. Within each run:

1. **Exported first**: exported methods, in their current order.
2. **Then call order**: the unexported methods the exported ones use, in the order they first use them, each followed by its own helpers, depth first. Then the other unexported methods that start something of their own, in their current order, each followed by its helpers: those used by a function or another type's methods, or by no method of their run. Uses from the same type's methods in other runs don't count.
3. **Shared helpers once**: a helper several methods use follows the first of them.

Each run's methods take the places the run already holds, so nothing moves out of its section: a file in feature sections, marked by declarations or by comments, keeps them, and a method used from another section stays in its own.

This means:
- In each run, public methods come first and helpers after them, top-down
- Types, constructors, functions and other declarations never move
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

**Note**: Like `go fmt`, gomsort processes directories recursively by default, with or without a `go.mod`. It skips `_test.go` files, generated files (those marked `// Code generated ... DO NOT EDIT.`), and directories the go command ignores: `testdata`, `vendor`, and names starting with `.` or `_`. A file it can't parse is reported on stderr and the others are still sorted; gomsort then exits with status 1.

gomsort parses with the `go/parser` of the Go that builds it, as gofmt does, so it knows the syntax of that Go version. The release binaries are built with Go 1.27; `go install` builds with your Go, so use Go 1.27 or later for code with methods that have type parameters.

### As a check in CI

`-n` prints a line for each file it would change and nothing otherwise, so a check can fail on any output:

```bash
out=$(gomsort -n .) && [ -z "$out" ] || { echo "$out"; exit 1; }
```

### As an analyzer

`github.com/borovikovd/gomsort/pkg/analyzer` provides a `go/analysis` analyzer named `msort` that reports files whose methods would be reordered, for use with `singlechecker`, `multichecker` or a custom golangci-lint build. It isn't one of golangci-lint's built-in linters.

## Scale

gomsort parses with Go's own `go/parser` and moves declarations as text, so a dry run costs little more than parsing. It handles one file at a time and keeps nothing between files, so memory follows the largest file, not the size of the codebase. Generated files, and files where no line starts with `func (` (no method, as gofmt writes one), are skipped before parsing. Each file takes time linear in its size, apart from sorting its methods (`O(m log m)` for `m` methods).

On an M-series Mac, a dry run on Kubernetes (13,500 Go files, 2.3 million lines outside tests) takes about 1.4 seconds and 20 MB, and on Prometheus about 0.2 seconds.

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
- Go 1.21 or later; the go command fetches the Go 1.27 toolchain the module asks for
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
2. **Find Uses**: Record which methods of its run each method uses through its receiver, by calling them or passing them on as values (`s.connect()`, `run(s.serve)`), in the order it first uses them. Uses from the same type's other runs don't count. Any other `x.name` in the file, in a function or a method, counts as a use from outside, which keeps a method in its place. Matching by name may mistake another type's method or a field for one of ours, which can only keep a method where it is. Recursion doesn't count.
3. **Order**: For each run, place its exported methods, then its unexported methods in call order, depth first
4. **Rewrite**: Put each run's methods back in the places the run held. Each method's text moves as it is, from the end of the previous declaration to the end of its own last line, so its doc comment and anything before it come along; the file is then gofmt'd.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by code organization principles from Clean Code and other software engineering best practices
- Built using Go's excellent AST and static analysis packages