package handlers

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const chatbotServiceURL = "http://localhost:8050"

// ProxyChatMessage forwards chat messages to the chatbot service
func ProxyChatMessage(c *gin.Context) {
	proxyRequest(c, chatbotServiceURL+"/chat", "POST")
}

// ProxyChatGreeting forwards greeting requests to the chatbot service
func ProxyChatGreeting(c *gin.Context) {
	proxyRequest(c, chatbotServiceURL+"/greeting", "GET")
}

// ProxyChatHealth forwards health check requests to the chatbot service
func ProxyChatHealth(c *gin.Context) {
	proxyRequest(c, chatbotServiceURL+"/health", "GET")
}

// proxyRequest handles proxying requests to the chatbot service
func proxyRequest(c *gin.Context, targetURL string, method string) {
	client := &http.Client{}

	var req *http.Request
	var err error

	if method == "POST" {
		// Read the request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
			return
		}
		req, err = http.NewRequest(method, targetURL, bytes.NewBuffer(body))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
			return
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, targetURL, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
			return
		}
	}

	// Forward the request
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Chatbot service unavailable",
			"details": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Forward the response
	c.Data(resp.StatusCode, "application/json", respBody)
}
