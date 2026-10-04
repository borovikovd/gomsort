module github.com/borovikovd/gomsort

go 1.26.0

// gomsort parses with the go/parser of the Go that builds it, so it builds
// with a recent one to know recent syntax, such as methods with type
// parameters (Go 1.27).
toolchain go1.27.1

require golang.org/x/tools v0.51.0
