package server

import (
	"github.com/gin-gonic/gin"
	"kvstore/internal/store" // Temporary store, will be replaced by an interface
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	store  *store.InMemoryStore // Temporary store
	router *gin.Engine
}

// New creates a new Server instance.
func New(storage *store.InMemoryStore) *Server {
	router := gin.Default()

	s := &Server{
		store:  storage,
		router: router,
	}

	s.registerRoutes()

	return s
}

// registerRoutes registers all the API routes.
func (s *Server) registerRoutes() {
	v1 := s.router.Group("/kv")
	{
		v1.GET("/:key", s.getHandler)
		v1.PUT("/:key", s.putHandler)
		v1.DELETE("/:key", s.deleteHandler)
		v1.GET("", s.getAllHandler)
	}
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}