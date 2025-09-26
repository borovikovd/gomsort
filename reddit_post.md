# Built a Go tool that intelligently sorts methods by call depth and usage patterns

I got tired of scrolling through poorly organized Go methods, so I built **gomsort** - a tool that automatically sorts methods within types using call graph analysis.

**The Problem**: Methods in Go structs are often randomly ordered, making code hard to follow. You end up with public entry points scattered between private helpers.

**The Solution**: gomsort analyzes your call graphs and sorts methods by:
1. Public methods first 
2. Entry points (low call depth) before helpers
3. Shared utilities (high in-degree) at the bottom

**Example transformation:**
```go
// Before - random order
func (s *Server) helper() string { return "help" }
func (s *Server) Start() error { return s.connect() }
func (s *Server) connect() error { s.helper(); return nil }
func (s *Server) Stop() error { return nil }

// After - logical flow
func (s *Server) Start() error { return s.connect() }
func (s *Server) Stop() error { return nil }
func (s *Server) connect() error { s.helper(); return nil }
func (s *Server) helper() string { return "help" }
```

**Features:**
- Works like `go fmt` - recursive by default
- Integrates with golangci-lint
- Preserves comments and semantics
- Configurable via `.msort.json`

Install: `go install github.com/borovikovd/gomsort@latest`

Repo: https://github.com/borovikovd/gomsort

What do you think? Would this help your codebases?