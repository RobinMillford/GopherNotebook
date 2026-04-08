package api

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware adds CORS headers for the Next.js frontend.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Strictly allow only the local frontend
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-LLM-Provider, X-LLM-Model")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
