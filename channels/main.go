package main

import (
	"fmt"
	"os"

	"github.com/slack-go/slack"
)

func listChannels(token string) {
	api := slack.New(token)

	// Call the conversations.list API method
	channels, cursor, err := api.GetConversations(&slack.GetConversationsParameters{
		Types: []string{"public_channel"}, // You can specify the types of channels to retrieve
	})

	if err != nil {
		fmt.Printf("Error listing channels: %s\n", err)
		return
	}

	for _, channel := range channels {
		fmt.Printf("Channel Name: %s, Channel ID: %s\n", channel.Name, channel.ID)
	}

	// You can also access the pagination cursor if needed
	fmt.Printf("Cursor: %s\n", cursor)
}

func main() {
	// Replace "your-token" with your Slack bot token
	//botToken := "your-token"
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		fmt.Println("SLACK_BOT_TOKEN=None")
		os.Exit(1)
	}
	listChannels(botToken)
}
