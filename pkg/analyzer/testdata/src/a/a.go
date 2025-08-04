package a

type Server struct{}

func (s *Server) Start() error {
	s.helper()
	return nil
}

func (s *Server) helper() string {
	return "help"
}
// want "methods in this file could be better sorted for readability"
// Methods are in wrong order - helper before entry points
