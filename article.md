# Building a Go Tool That Understands Your Code: Call Graph Analysis for Method Sorting

*How I solved the problem of poorly organized Go methods with algorithmic thinking*

---

Have you ever opened a Go file and found yourself scrolling endlessly to understand the flow? Public methods scattered between private helpers, entry points buried after implementation details, and utilities mixed in randomly? I certainly have, and it drove me to build something about it.

## The Problem: Chaos in Code Organization

Consider this typical Go struct:

```go
type Server struct {
    addr   string
    client *http.Client
}

func (s *Server) validateConfig() error {
    // validation logic
    return nil
}

func (s *Server) Start() error {
    if err := s.validateConfig(); err != nil {
        return err
    }
    return s.connect()
}

func (s *Server) logStatus(msg string) {
    fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), msg)
}

func (s *Server) connect() error {
    s.logStatus("connecting...")
    // connection logic
    return nil
}

func (s *Server) Stop() error {
    s.logStatus("stopping...")
    return nil
}
```

What's wrong here? The methods are organized randomly:
1. `validateConfig()` - a private helper
2. `Start()` - a public entry point that calls both helpers
3. `logStatus()` - a shared utility
4. `connect()` - another private helper
5. `Stop()` - another public entry point

To understand the flow, you have to jump around the file. Not ideal.

## The Solution: Think Like a Call Graph

What if we could automatically organize methods based on how they actually relate to each other? That's where **call graph analysis** comes in.

A call graph represents the calling relationships between functions. By analyzing these relationships, we can determine:

- **Entry points**: Methods called from outside (low call depth)
- **Helpers**: Methods called by other methods (higher call depth)  
- **Utilities**: Methods called by many others (high in-degree)

## Introducing gomsort

I built [gomsort](https://github.com/borovikovd/gomsort) - a tool that sorts Go methods intelligently using this approach.

### The Algorithm

Methods are sorted by these criteria (in order):

1. **Receiver Type** - Group methods by their receiver (alphabetical)
2. **Visibility** - Exported methods before private ones
3. **Call Depth** - Entry points (shallow) before deep helpers
4. **In-Degree** - Shared utilities (many callers) appear last
5. **Original Position** - Stable sort fallback

### The Result

Here's our server example after gomsort:

```go
type Server struct {
    addr   string
    client *http.Client
}

// Public entry points first
func (s *Server) Start() error {
    if err := s.validateConfig(); err != nil {
        return err
    }
    return s.connect()
}

func (s *Server) Stop() error {
    s.logStatus("stopping...")
    return nil
}

// Private helpers in call order
func (s *Server) connect() error {
    s.logStatus("connecting...")
    // connection logic
    return nil
}

func (s *Server) validateConfig() error {
    // validation logic
    return nil
}

// Shared utilities last
func (s *Server) logStatus(msg string) {
    fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), msg)
}
```

Much better! Now you can read top-to-bottom: public interface first, implementation details follow logically.

## How It Works

### 1. AST Parsing

```go
fset := token.NewFileSet()
node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
```

We parse each Go file into an Abstract Syntax Tree, preserving comments and formatting.

### 2. Call Graph Building

```go
func (cg *CallGraph) AddCall(caller, callee string) {
    if cg.Graph[caller] == nil {
        cg.Graph[caller] = make(map[string]bool)
    }
    cg.Graph[caller][callee] = true
    cg.InDegree[callee]++
}
```

We walk the AST to find method calls and build a directed graph of dependencies.

### 3. Depth Calculation

```go
func (cg *CallGraph) CalculateMaxDepth(method string, visited map[string]bool) int {
    if visited[method] {
        return 0 // Cycle detection
    }
    
    maxDepth := 0
    visited[method] = true
    
    for callee := range cg.Graph[method] {
        depth := 1 + cg.CalculateMaxDepth(callee, visited)
        if depth > maxDepth {
            maxDepth = depth
        }
    }
    
    delete(visited, method)
    return maxDepth
}
```

We calculate the maximum call depth for each method using recursive traversal.

### 4. Smart Sorting

```go
func (s *Sorter) shouldSwap(i, j MethodInfo) bool {
    // Different receivers? Alphabetical order
    if i.ReceiverType != j.ReceiverType {
        return i.ReceiverType > j.ReceiverType
    }
    
    // Same receiver? Exported first
    if i.IsExported != j.IsExported {
        return !i.IsExported
    }
    
    // Same visibility? Lower depth first
    if i.MaxDepth != j.MaxDepth {
        return i.MaxDepth > j.MaxDepth
    }
    
    // Same depth? Higher in-degree last
    if i.InDegree != j.InDegree {
        return i.InDegree < j.InDegree
    }
    
    // Fallback to original position
    return i.Position > j.Position
}
```

The sorting logic implements our prioritization strategy while maintaining stability.

## Usage & Features

### Installation

```bash
go install github.com/borovikovd/gomsort@latest
```

### Basic Usage

```bash
# Sort a single file
gomsort server.go

# Sort entire project (recursive)
gomsort .

# Dry run to preview changes
gomsort -n .
```

### Integration with golangci-lint

Add to your `.golangci.yml`:

```yaml
linters:
  enable:
    - msort
```

### Configuration

Create `.msort.json` for customization:

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

## Real-World Impact

After running gomsort on several codebases, I've noticed:

- **Faster Code Reviews**: Reviewers spend less time understanding method relationships
- **Easier Debugging**: Entry points are immediately visible at the top
- **Better Onboarding**: New team members can follow code flow more intuitively
- **Consistent Style**: Like `gofmt` but for logical organization

## The Technical Challenge

Building this tool involved several interesting problems:

### Cycle Detection
Method calls can form cycles (A calls B, B calls A). We handle this gracefully by tracking visited nodes during depth calculation.

### Comment Preservation
Go's AST discards some formatting information. We use the [dst](https://github.com/dave/dst) package to preserve comments and spacing.

### Performance
For large codebases, we optimize by:
- Parsing files in parallel
- Caching call graph analysis
- Only rewriting files that actually change

## What's Next?

Some ideas for future improvements:

- **Interface awareness**: Sort implementations near their interfaces
- **Test integration**: Keep tests close to the methods they test
- **IDE plugins**: Real-time sorting suggestions
- **Metrics**: Measure complexity reduction

## Try It Yourself

Want to see how your codebase looks after intelligent method sorting?

```bash
# Install
go install github.com/borovikovd/gomsort@latest

# Try it (dry run first!)
gomsort -n .
```

Check out the [repository](https://github.com/borovikovd/gomsort) for more details, examples, and contribution guidelines.

## Conclusion

Code organization isn't just about aesthetics - it's about reducing cognitive load and making maintenance easier. By applying algorithmic thinking to a common problem, we can automate what was previously a manual and subjective task.

gomsort represents a small step toward more intelligent developer tooling. Just as `gofmt` standardized formatting, tools like this can standardize logical organization.

What do you think? Would call graph-based method sorting help your projects? Let me know in the comments!

---

*Follow me for more posts about Go tooling, algorithms, and developer productivity. You can also check out [gomsort on GitHub](https://github.com/borovikovd/gomsort).*

---

**Tags:** #go #golang #developer-tools #static-analysis #code-organization #algorithms