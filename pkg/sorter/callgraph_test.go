package sorter

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
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

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	cg := buildCallGraph(fset, file)
	methods := cg.GetMethods()

	if len(methods) != 5 {
		t.Errorf("Expected 5 methods, got %d", len(methods))
	}

	methodMap := make(map[string]*MethodInfo)
	for _, method := range methods {
		methodMap[method.Name] = method
	}

	tests := []struct {
		methodName string
		callees    string
		callers    int
	}{
		{"Start", "connect", 0},
		{"connect", "authenticate", 2}, // used by Start and Status
		{"authenticate", "", 1},
		{"Stop", "", 0},
		{"Status", "connect", 0},
	}
	for _, test := range tests {
		method := methodMap[test.methodName]
		if method == nil {
			t.Fatalf("method %s not found", test.methodName)
		}
		var callees []string
		for _, c := range method.Callees {
			callees = append(callees, c.Name)
		}
		if got := strings.Join(callees, " "); got != test.callees {
			t.Errorf("%s uses %q, want %q", test.methodName, got, test.callees)
		}
		if method.Callers != test.callers {
			t.Errorf("%s: %d callers, want %d", test.methodName, method.Callers, test.callers)
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

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	cg := buildCallGraph(fset, file)
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

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	cg := buildCallGraph(fset, file)
	methods := cg.GetMethods()

	if len(methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(methods))
	}

	// Each uses the other, so neither is an entry point; both are still
	// placed, in their order.
	var names []string
	for _, m := range sortMethods(methods) {
		names = append(names, m.Name)
	}
	if got := strings.Join(names, " "); got != "methodA methodB" {
		t.Errorf("got %q, want methodA methodB", got)
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
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	methods := map[string]*MethodInfo{}
	for _, m := range buildCallGraph(fset, file).GetMethods() {
		methods[m.Name] = m
	}
	if got := methods["visit"].Callers; got != 1 {
		t.Errorf("visit: %d callers, want 1 (Walk, counted once; recursion ignored)", got)
	}
	if got := len(methods["Walk"].Callees); got != 1 {
		t.Errorf("Walk uses %d methods, want 1", got)
	}
}

func TestCallGraphMarksUsesFromOutsideByName(t *testing.T) {
	// Uses through anything but the method's own receiver count as outside
	// uses of every method with that name: here add and quote, through a
	// variable, a composite literal and a field. Only uses through the
	// receiver link a method to its helpers.
	source := `
package test

type ddlGen struct{}

func (g ddlGen) alter() []string { g.recreate(); return ddlGen{}.add() }
func (g ddlGen) recreate()        { g.drop() }
func (g ddlGen) add() []string    { return nil }
func (g ddlGen) drop()            {}
func (g ddlGen) quote() string    { return "" }

type wrapper struct{ gen ddlGen }

func ChangeDDL(w wrapper) {
	g := ddlGen{}
	g.add()
	w.gen.quote()
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	methods := map[string]*MethodInfo{}
	for _, m := range buildCallGraph(fset, file).GetMethods() {
		methods[m.Name] = m
	}
	for name, outside := range map[string]bool{"alter": false, "recreate": false, "add": true, "drop": false, "quote": true} {
		if got := methods[name].UsedOutside; got != outside {
			t.Errorf("%s: used outside %v, want %v", name, got, outside)
		}
	}
	if got := len(methods["alter"].Callees); got != 1 {
		t.Errorf("alter uses %d methods through its receiver, want 1 (recreate)", got)
	}
}

func TestAddCallAcrossTypesMarksOutsideUse(t *testing.T) {
	cg := NewCallGraph()
	helper := &MethodInfo{Name: "helper", ReceiverName: "Server"}
	cg.AddMethod(helper)
	cg.AddCall("Client", "Connect", "Server", "helper")
	if !helper.UsedOutside || helper.Callers != 0 {
		t.Errorf("used outside %v, %d callers; want true, 0", helper.UsedOutside, helper.Callers)
	}
}
