package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	timeoutedClient *http.Client
	baseUrl         string
	apiKey          string
	model           string
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(openaiApiKey string, model string) *Client {
	return &Client{
		// Model responses can take a while, so allow more time than the other clients
		timeoutedClient: &http.Client{Timeout: 60 * time.Second},
		baseUrl:         "https://api.openai.com/v1",
		apiKey:          openaiApiKey,
		model:           model,
	}
}

// Ask sends a single user prompt and returns the model's text reply.
func (client *Client) Ask(prompt string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    client.model,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, client.baseUrl+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+client.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.timeoutedClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: request failed: %w", err)
	}
	defer res.Body.Close()

	resp := chatResponse{}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", fmt.Errorf("openai: decoding response: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("openai: %s (status %d)", resp.Error.Message, res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai: unexpected status %d", res.StatusCode)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai: response had no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
