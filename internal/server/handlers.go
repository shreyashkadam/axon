package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// KeyValue represents the JSON payload for a PUT request.
type KeyValue struct {
	Value string `json:"value" binding:"required"`
}

// putHandler handles storing a key-value pair.
func (s *Server) putHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	var kv KeyValue
	if err := c.ShouldBindJSON(&kv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.store.Put([]byte(key), []byte(kv.Value)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// getHandler handles retrieving a value by its key.
func (s *Server) getHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	value, err := s.store.Get([]byte(key))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key, "value": string(value)})
}

// deleteHandler handles removing a key-value pair.
func (s *Server) deleteHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	if err := s.store.Delete([]byte(key)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// getAllHandler retrieves all key-value pairs from the local store.
func (s *Server) getAllHandler(c *gin.Context) {
	values, err := s.store.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]string)
	for k, v := range values {
		result[k] = string(v)
	}

	c.JSON(http.StatusOK, result)
}