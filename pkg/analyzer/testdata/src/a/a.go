package a // want "methods in this file could be better sorted for readability"

type Server struct{}

// Methods are in wrong order - helper before entry points
func (s *Server) helper() string {
	return "help"
}

func (s *Server) Start() error {
	s.helper()
	return nil
}
