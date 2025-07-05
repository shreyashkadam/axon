package server

import (
	"github.com/gin-gonic/gin"
	"kvstore/internal/cluster"
	"kvstore/internal/store"
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	store     store.Store
	node      *cluster.Node
	router    *gin.Engine
	isCluster bool
}

// New creates a new Server instance.
// If the node is nil, it runs in single-node mode.
func New(storage store.Store, node *cluster.Node) *Server {
	router := gin.Default()

	s := &Server{
		store:     storage,
		node:      node,
		router:    router,
		isCluster: (node != nil),
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

	// Internal API for node-to-node communication
	if s.isCluster {
		// Will be implemented later
	}
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}