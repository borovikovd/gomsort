package sorter

import (
	"testing"

	"github.com/dave/dst/decorator"
)

func TestCallGraphBuilding(t *testing.T) {
	source := `
package test

type Server struct{}

func (s *Server) Start() error {
	return s.connect()
}

func (s *Server) connect() error {
	return s.authenticate()
}

func (s *Server) authenticate() error {
	return nil
}

func (s *Server) Stop() error {
	return nil
}

func (s *Server) Status() string {
	if s.connect() != nil {
		return "disconnected"
	}
	return "connected"
}
`

	file, err := decorator.Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	cg := buildCallGraph(file)
	methods := cg.GetMethods()

	if len(methods) != 5 {
		t.Errorf("Expected 5 methods, got %d", len(methods))
	}

	methodMap := make(map[string]*MethodInfo)
	for _, method := range methods {
		methodMap[method.Name] = method
	}

	tests := []struct {
		methodName     string
		expectedDepth  int
		expectedDegree int
	}{
		{"Start", 0, 0},        // an entry point: nothing calls it
		{"connect", 1, 2},      // called by Start and Status, both entry points
		{"authenticate", 2, 1}, // called by connect, one call below them
		{"Stop", 0, 0},         // an entry point
		{"Status", 0, 0},       // an entry point
	}

	for _, test := range tests {
		method, exists := methodMap[test.methodName]
		if !exists {
			t.Errorf("Method %s not found", test.methodName)
			continue
		}

		if method.MaxDepth != test.expectedDepth {
			t.Errorf("Method %s: expected depth %d, got %d", test.methodName, test.expectedDepth, method.MaxDepth)
		}

		if method.InDegree != test.expectedDegree {
			t.Errorf("Method %s: expected in-degree %d, got %d", test.methodName, test.expectedDegree, method.InDegree)
		}
	}
}

func TestCallGraphWithMultipleReceivers(t *testing.T) {
	source := `
package test

type Client struct{}
type Server struct{}

func (c *Client) Connect() error {
	return c.dial()
}

func (c *Client) dial() error {
	return nil
}

func (s *Server) Start() error {
	return s.listen()
}

func (s *Server) listen() error {
	return nil
}
`

	file, err := decorator.Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	cg := buildCallGraph(file)
	methods := cg.GetMethods()

	if len(methods) != 4 {
		t.Errorf("Expected 4 methods, got %d", len(methods))
	}

	// Group methods by receiver
	clientMethods := make([]*MethodInfo, 0)
	serverMethods := make([]*MethodInfo, 0)

	for _, method := range methods {
		if method.ReceiverName == "Client" {
			clientMethods = append(clientMethods, method)
		} else if method.ReceiverName == "Server" {
			serverMethods = append(serverMethods, method)
		}
	}

	if len(clientMethods) != 2 {
		t.Errorf("Expected 2 Client methods, got %d", len(clientMethods))
	}

	if len(serverMethods) != 2 {
		t.Errorf("Expected 2 Server methods, got %d", len(serverMethods))
	}
}

func TestCallGraphCycleDetection(t *testing.T) {
	source := `
package test

type Server struct{}

func (s *Server) methodA() error {
	return s.methodB()
}

func (s *Server) methodB() error {
	return s.methodA() // Creates a cycle
}
`

	file, err := decorator.Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	cg := buildCallGraph(file)
	methods := cg.GetMethods()

	if len(methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(methods))
	}

	// In case of cycles, the algorithm should handle it gracefully
	// and not infinite loop
	for _, method := range methods {
		if method.MaxDepth < 0 {
			t.Errorf("Method %s has negative depth: %d", method.Name, method.MaxDepth)
		}
	}
}

func TestMethodKey(t *testing.T) {
	tests := []struct {
		receiver string
		method   string
		expected string
	}{
		{"Server", "Start", "Server.Start"},
		{"Client", "Connect", "Client.Connect"},
		{"", "function", ".function"},
	}

	for _, test := range tests {
		result := methodKey(test.receiver, test.method)
		if result != test.expected {
			t.Errorf("methodKey(%s, %s) = %s, want %s", test.receiver, test.method, result, test.expected)
		}
	}
}

func TestCallGraphCountsDistinctCallersAndIgnoresRecursion(t *testing.T) {
	source := `
package test

type Tree struct{}

func (t *Tree) Walk() {
	t.visit()
	t.visit()
}

func (t *Tree) visit() {
	t.visit()
}
`
	file, err := decorator.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	methods := map[string]*MethodInfo{}
	for _, m := range buildCallGraph(file).GetMethods() {
		methods[m.Name] = m
	}
	if got := methods["visit"].InDegree; got != 1 {
		t.Errorf("visit: in-degree %d, want 1 (Walk, counted once; recursion ignored)", got)
	}
	if got := methods["Walk"].MaxDepth; got != 0 {
		t.Errorf("Walk: depth %d, want 0", got)
	}
	if got := methods["visit"].MaxDepth; got != 1 {
		t.Errorf("visit: depth %d, want 1", got)
	}
}
