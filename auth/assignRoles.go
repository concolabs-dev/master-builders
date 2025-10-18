package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

func GetManagementToken() (string, error) {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	auth0Domain := os.Getenv("AUTH0_MANAGEMENT_DOMAIN")
	clientID := os.Getenv("AUTH0_MANAGEMENT_CLIENT_ID")
	clientSecret := os.Getenv("AUTH0_MANAGEMENT_CLIENT_SECRET")
	audience := "https://" + auth0Domain + "/api/v2/"

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("audience", audience)

	resp, err := http.PostForm("https://"+auth0Domain+"/oauth/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get access token")
	}
	return token, nil
}

func AssignRole(userID string, roleID string, token string) error {
	auth0Domain := os.Getenv("AUTH0_MANAGEMENT_DOMAIN")

	url := fmt.Sprintf("https://%s/api/v2/users/%s/roles", auth0Domain, userID)

	payload := map[string]interface{}{
		"roles": []string{roleID},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		// Log response body for debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign role: %s", string(bodyBytes))
	}
	return nil
}
