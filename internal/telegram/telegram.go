package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type UpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type Update struct {
	Message *Message `json:"message"`
	Id      int      `json:"update_id"`
}

type Message struct {
	Text string `json:"text"`
	Chat *Chat  `json:"chat"`
	User *User  `json:"from"`
}

type Chat struct {
	Id int `json:"id"`
}

type User struct {
	Id int `json:"id"`
}

type Client struct {
	timeoutedClient *http.Client
	baseUrl         string
	updateOffset    int
}

func NewClient(token string) *Client {
	return &Client{
		timeoutedClient: &http.Client{Timeout: time.Second * 30},
		baseUrl:         "https://api.telegram.org/bot" + token,
		updateOffset:    0,
	}
}

func (client *Client) GetUpdates() (UpdatesResponse, error) {

	res, getErr := client.timeoutedClient.Get(client.baseUrl + "/getUpdates?offset=" + fmt.Sprint(client.updateOffset) + "&timeout=20")
	if getErr != nil {
		fmt.Println("Error calling telegram API")
		return UpdatesResponse{}, getErr
	}
	defer res.Body.Close()

	resp := UpdatesResponse{}
	decodingErr := json.NewDecoder(res.Body).Decode(&resp)
	if decodingErr != nil {
		fmt.Println("Error decoding telegram API response")
		return UpdatesResponse{}, decodingErr
	}
	fmt.Println("Updates:", resp.Result)
	if len(resp.Result) > 0 {
		client.UpdateOffset(resp.Result[len(resp.Result)-1].Id + 1)
	}
	return resp, nil
}

func (client *Client) SendMessage(text string, chatId int) error {
	res, err := client.timeoutedClient.PostForm(client.baseUrl+"/sendMessage", url.Values{
		"chat_id": {fmt.Sprint(chatId)},
		"text":    {text},
	})

	if err != nil {
		return fmt.Errorf("sendMessage: request failed")
	}
	defer res.Body.Close()
	return nil
}

func IsUserAllowed(userId int, allowedUserId int) bool {
	// TODO evolve for a team
	return userId == allowedUserId
}

func (c *Client) UpdateOffset(newOffset int) {
	c.updateOffset = newOffset
}
