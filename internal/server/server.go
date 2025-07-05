package server

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"kvstore/internal/cluster"
	"kvstore/internal/store"
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	store       *store.PersistentStore
	node        *cluster.Node
	router      *gin.Engine
	isCluster   bool
	consistency string
}

// New creates a new Server instance.
// If the node is nil, it runs in single-node mode.
func New(storage *store.PersistentStore, node *cluster.Node, consistent string) *Server {
	router := gin.Default()

	// Add CORS middleware to allow the UI to connect
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	s := &Server{
		store:       storage,
		node:        node,
		router:      router,
		isCluster:   (node != nil),
		consistency: consistent,
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
			internal.POST("/operation", s.internalOperationHandler)
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