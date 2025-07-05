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
	// Public API for key-value operations
	v1 := s.router.Group("/kv")
	{
		v1.GET("/:key", s.getHandler)
		v1.PUT("/:key", s.putHandler)
		v1.DELETE("/:key", s.deleteHandler)
		v1.GET("", s.getAllHandler)
	}

	// Internal API for node-to-node communication
	if s.isCluster {
		internal := s.router.Group("/internal")
		{
			internal.GET("/get/:key", s.internalGetHandler)
			internal.GET("/status", s.internalStatusHandler)
			internal.GET("/version/:key", s.internalVersionHandler)

			raftGroup := internal.Group("/raft")
			{
				raftGroup.POST("/join", s.internalRaftJoinHandler)
				raftGroup.POST("/remove", s.internalRaftRemoveHandler)
			}
		}
	}
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}