package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/magdapoppins/ceo/internal/linear"
	"github.com/magdapoppins/ceo/internal/openai"
	"github.com/magdapoppins/ceo/internal/telegram"
)

func main() {
	telegramApiKey, exists := os.LookupEnv("TELEGRAM_API_KEY")
	if !exists {
		panic("Environment variable 'TELEGRAM_API_KEY' is not set")
	}
	linearApiKey, exists := os.LookupEnv("LINEAR_API_KEY")
	if !exists {
		panic("Environment variable 'LINEAR_API_KEY' is not set")
	}
	openAiApiKey, exists := os.LookupEnv("OPENAI_API_KEY")
	if !exists {
		panic("Environment variable 'OPENAI_API_KEY' is not set")
	}

	allowedUserIdStr, exists := os.LookupEnv("TELEGRAM_ALLOWED_USER_ID")
	allowedUserId, err := strconv.Atoi(allowedUserIdStr)
	if err != nil {
		fmt.Println("Error converting TELEGRAM_ALLOWED_USER_ID to int:", err)
		// the string wasn't a valid number, e.g. "abc" or ""
	}

	tgClient := telegram.NewClient(telegramApiKey)
	lnClient := linear.NewClient(linearApiKey)
	openAiClient := openai.NewClient(openAiApiKey, "gpt-6-luna")

	for {
		resp, err := tgClient.GetUpdates()
		if err != nil {
			fmt.Println("Error getting updates from telegram API. Retrying in 5 seconds...")
			time.Sleep(5 * time.Second) // Wait a bit, then try again
			continue
		}
		fmt.Println("Response:", resp)

		for _, update := range resp.Result {
			switch update.Message.Text {
			case "/assign":
				dailyBriefing(tgClient, lnClient, openAiClient, allowedUserId)
			default:
				if update.Message != nil {
					if !telegram.IsUserAllowed(update.Message.User.Id, allowedUserId) {
						fmt.Println("User not allowed:", update.Message.User.Id)
						continue
					}
					err := tgClient.SendMessage(`Available commands: /assign`, update.Message.Chat.Id)
					if err != nil {
						fmt.Println("Error sending message to telegram API")
					}
				}
			}
		}
	}
}

func dailyBriefing(tgClient *telegram.Client, lnClient *linear.Client, openAiClient *openai.Client, allowedUserId int) {
	issues, err := lnClient.GetIssues()
	if err != nil {
		fmt.Println("Error fetching issues from Linear API")
		return
	}
	openAiPrompt := `You are my strict boss who is assigning work to me today. 
	I can do 1 task at a time. Please summarize the following issues, and pick one or two highest priority ones to focus on today:\n` + issues + `\n
	Write the response in a telegram message format. Describe the tasks briefly and include issue title and some helpful links for me.
	to get started. Do not write anything else.`
	summary, err := openAiClient.Ask(openAiPrompt)
	if err != nil {
		fmt.Println("Error getting summary from OpenAI API")
		return
	}
	err = tgClient.SendMessage(summary, allowedUserId)
	if err != nil {
		fmt.Println("Error sending message to telegram API")
	}
}
