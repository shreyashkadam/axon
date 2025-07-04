package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// KeyValue represents the JSON payload for a PUT request.
type KeyValue struct {
	Value string `json:"value" binding:"required"`
}

func (s *Server) putHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "put placeholder"})
}

func (s *Server) getHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "get placeholder"})
}

func (s *Server) deleteHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "delete placeholder"})
}

func (s *Server) getAllHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "get all placeholder"})
}