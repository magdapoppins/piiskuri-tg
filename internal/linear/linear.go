package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const issuesQuery = `
query Issues($after: String) {
	issues(first: 50, after: $after) {
		nodes {
			identifier
			title
			description
			priority
			state { name }
			labels { nodes { name } }
			project { name }
		}
		pageInfo { hasNextPage endCursor }
	}
}`

type Client struct {
	timeoutedClient *http.Client
	baseUrl         string
	apiKey          string
}

// Typed issues, for when you want to work with fields in Go instead of passing raw JSON on.
// type Issue struct {
// 	Identifier  string `json:"identifier"`
// 	Title       string `json:"title"`
// 	Description string `json:"description"`
// 	Priority    int    `json:"priority"` // 0 = none, 1 = urgent ... 4 = low
// 	State       struct {
// 		Name string `json:"name"`
// 	} `json:"state"`
// 	Labels struct {
// 		Nodes []struct {
// 			Name string `json:"name"`
// 		} `json:"nodes"`
// 	} `json:"labels"`
// 	Project *struct {
// 		Name string `json:"name"`
// 	} `json:"project"` // nil when the issue isn't in a project
// }

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type issuesResponse struct {
	Data struct {
		Issues struct {
			// Nodes []Issue `json:"nodes"`
			Nodes    json.RawMessage `json:"nodes"` // kept as raw JSON text, not decoded
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"issues"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func NewClient(linearApiKey string) *Client {
	return &Client{
		timeoutedClient: &http.Client{Timeout: 30 * time.Second},
		baseUrl:         "https://api.linear.app/graphql",
		apiKey:          linearApiKey,
	}
}

// GetIssues fetches every issue the API key can see, one page at a time,
// and returns them as JSON text: one array of issues per page, one page per line.
func (client *Client) GetIssues() (string, error) {
	// var issues []Issue
	var issues strings.Builder
	var after any // nil on the first page, which becomes JSON null

	for {
		resp, err := client.fetchPage(after)
		if err != nil {
			return "", err
		}
		// issues = append(issues, resp.Data.Issues.Nodes...)
		issues.Write(resp.Data.Issues.Nodes)
		issues.WriteString("\n")

		if !resp.Data.Issues.PageInfo.HasNextPage {
			return issues.String(), nil
		}
		after = resp.Data.Issues.PageInfo.EndCursor
	}
}

// fetchPage is a separate function so defer closes each response body before the next request.
func (client *Client) fetchPage(after any) (issuesResponse, error) {
	var resp issuesResponse

	body, err := json.Marshal(graphQLRequest{
		Query:     issuesQuery,
		Variables: map[string]any{"after": after},
	})
	if err != nil {
		return resp, err
	}

	req, err := http.NewRequest(http.MethodPost, client.baseUrl, bytes.NewReader(body))
	if err != nil {
		return resp, err
	}
	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.timeoutedClient.Do(req)
	if err != nil {
		return resp, fmt.Errorf("linear: request failed: %w", err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return resp, fmt.Errorf("linear: decoding response (status %d): %w", res.StatusCode, err)
	}
	if len(resp.Errors) > 0 {
		return resp, fmt.Errorf("linear: %s", resp.Errors[0].Message)
	}
	return resp, nil
}
