package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

var rdb *redis.Client

func main() {
	// Initialize Redis
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Assuming Redis is running locally in Docker
	})

	r := gin.Default()

	// Cache middleware: Check cache before forwarding to backend
	r.Use(CacheMiddleware(30 * time.Second)) // Cache TTL set to 30 seconds

	r.GET("/proxy/*url", func(c *gin.Context) {
		url := c.Param("url")

		// Simulate a backend call, proxying to 8040
		backendResponse := forwardToBackend(url)

		// Send the response back to the client
		c.JSON(http.StatusOK, backendResponse)
	})

	r.Run(":8080") // Proxy service listening on 8080
}

// CacheMiddleware: Check cache first, if no cache forward the request to the backend
func CacheMiddleware(ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// Create cache key based on request URI
		cacheKey := "cache:" + c.Request.RequestURI
		cachedResponse, err := rdb.Get(context.Background(), cacheKey).Result()
		if err == nil {
			// If cache exists, return the cached response
			c.Data(http.StatusOK, "application/json", []byte(cachedResponse))
			c.Abort()
			return
		}

		// No cache, proceed to backend and cache the response afterward
		writer := &bodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = writer

		// Continue to next handler
		c.Next()

		// Cache only successful responses (status 200)
		if c.Writer.Status() == http.StatusOK {
			rdb.Set(context.Background(), cacheKey, writer.body.String(), ttl)
		}
	}
}

// forwardToBackend forwards the request to the actual backend (running on port 8040)
func forwardToBackend(url string) string {
	// Here you can add logic to make an HTTP request to your backend server running on 8040
	// Example using http package to forward the request

	backendURL := fmt.Sprintf("http://localhost:8040%s", url)

	// Forward the request and get the response
	resp, err := http.Get(backendURL)
	if err != nil {
		return fmt.Sprintf("Error forwarding request: %v", err)
	}
	defer resp.Body.Close()

	// Read and return the response from backend
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response body: %v", err)
	}

	return buf.String()
}

// bodyWriter is a custom ResponseWriter to capture the response body
type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
