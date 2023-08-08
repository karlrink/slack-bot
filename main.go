package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    "time"
    "context"
    "net/http"
    "io/ioutil"
    "encoding/json"

    "github.com/slack-go/slack/slackevents"
    "github.com/slack-go/slack/socketmode"

    "github.com/slack-go/slack"

    openai "github.com/sashabaranov/go-openai"
)

func main() {
	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		panic("SLACK_APP_TOKEN must be set.\n")
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		panic("SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		panic("SLACK_BOT_TOKEN must be set.\n")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		panic("SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	openaiToken := os.Getenv("OPENAI_API_KEY")
	if botToken == "" {
		panic("OPENAI_API_KEY must be set.\n")
	}

	if !strings.HasPrefix(openaiToken, "sk-") {
		panic("OPENAI_API_KEY must have the prefix \"sk-\".")
	}

	api := slack.New(
		botToken,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	socketmodeHandler := socketmode.NewSocketmodeHandler(client)

	socketmodeHandler.Handle(socketmode.EventTypeConnecting, middlewareConnecting)
	socketmodeHandler.Handle(socketmode.EventTypeConnectionError, middlewareConnectionError)
	socketmodeHandler.Handle(socketmode.EventTypeConnected, middlewareConnected)

	//\\ EventTypeEventsAPI //\\
	// Handle all EventsAPI
	socketmodeHandler.Handle(socketmode.EventTypeEventsAPI, middlewareEventsAPI)

	// Handle a specific event from EventsAPI
	socketmodeHandler.HandleEvents(slackevents.AppMention, middlewareAppMentionEvent)

	//\\ EventTypeInteractive //\\
	// Handle all Interactive Events
	socketmodeHandler.Handle(socketmode.EventTypeInteractive, middlewareInteractive)

	// Handle a specific Interaction
	socketmodeHandler.HandleInteraction(slack.InteractionTypeBlockActions, middlewareInteractionTypeBlockActions)

	// Handle all SlashCommand
	socketmodeHandler.Handle(socketmode.EventTypeSlashCommand, middlewareSlashCommand)
	//socketmodeHandler.HandleSlashCommand("/rocket", middlewareSlashCommand)

	// socketmodeHandler.HandleDefault(middlewareDefault)

	socketmodeHandler.RunEventLoop()
}

func middlewareConnecting(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connecting... to Slack with Socket Mode...")
}

func middlewareConnectionError(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connection failed. Retrying later...")
}

func middlewareConnected(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connected to Slack with Socket Mode.")
}


// Declare a map to keep track of messages that have been responded to
var respondedMessages = make(map[string]bool)

func middlewareEventsAPI(evt *socketmode.Event, client *socketmode.Client) {
    fmt.Println("middlewareEventsAPI")
    eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
    if !ok {
        fmt.Printf("Ignored %+v\n", evt)
        return
    }

    fmt.Printf("Event received middlewareEventsAPI: %+v\n", eventsAPIEvent)

    client.Ack(*evt.Request)

    switch eventsAPIEvent.Type {
    case slackevents.CallbackEvent:
        innerEvent := eventsAPIEvent.InnerEvent
        switch ev := innerEvent.Data.(type) {

        case *slackevents.MessageEvent:
            if ev.ChannelType == "im" && ev.BotID == "" {
                fmt.Printf("Direct message in %v", ev.Channel)
                // Check if we have already responded to this message
                if _, exists := respondedMessages[ev.ClientMsgID]; !exists {

                    originalMessage := strings.ToLower(ev.Text) // Convert to lowercase

                    if strings.HasPrefix(originalMessage, "tell a dad joke in channel") {

                        //channelID := strings.TrimPrefix(originalMessage, "tell a dad joke in channel <#")
                        channelID := strings.TrimPrefix(ev.Text, "tell a dad joke in channel <#")
                        channelID = strings.Split(channelID, "|")[0] // Assuming the channel mention format is <#CHANNEL_ID|name>

                        jokeText, jokeErr := getDadJoke()
                        if jokeErr != nil {
                            jokeText = "This is Not a Joke! " + jokeErr.Error()
                        }

                        _, _, err := client.Client.PostMessage(channelID, slack.MsgOptionText(jokeText, false))
                        if err != nil {
                            fmt.Printf("failed posting message: %v", err)
                        }

                    } else {


                        switch originalMessage {

                        case "dadjoke", "tell me a dadjoke", "tell me another dadjoke":
                            //response := "Yes, I can dad that"

                            jokeText, jokeErr := getDadJoke()
                            if jokeErr != nil {
                                jokeText = "This is Not a Joke! " + jokeErr.Error()
                            }

                            _, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText(jokeText, false))
                            if err != nil {
                                fmt.Printf("failed posting message: %v", err)
                            }

                        case "what is the weather like":
                            response := "I'm sorry, I can't provide weather information."
                            _, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText(response, false))
                            if err != nil {
                                fmt.Printf("failed posting message: %v", err)
                            }

                        case "how old are you":
                            response := "I'm just a computer program, I don't have an age."
                            _, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText(response, false))
                            if err != nil {
                                fmt.Printf("failed posting message: %v", err)
                            }

                        case "who are you":
                            response := "I am a chatbot designed to assist you with various tasks."
                            _, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText(response, false))
                            if err != nil {
                                fmt.Printf("failed posting message: %v", err)
                            }

                        case "openai":
                            //response := "I am a chatbot designed to assist you with various tasks."
                            openaiResponse, openaiErr := getOpenAIResponse(ev.Text)
                            if openaiErr != nil {
                                openaiResponse = "ResponseError: " + openaiErr.Error()
                            }

                            _, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText(openaiResponse, false))
                            if err != nil {
                                fmt.Printf("failed posting message: %v", err)
                            }

                        default:
                            //response := fmt.Sprintf("Howdy, i got your message: %s", ev.Text)
                            openaiResponse, openaiErr := getOpenAIResponse(ev.Text)
                            if openaiErr != nil {
                                openaiResponse = "ResponseError: " + openaiErr.Error()
                            }

                            _, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText(openaiResponse, false))
                            if err != nil {
                                fmt.Printf("failed posting message: %v", err)
                            }

                        } // send-switch-case

                    } //end-if-else

                    // Mark the message as responded in the map
                    respondedMessages[ev.ClientMsgID] = true
                }
            }

        case *slackevents.AppMentionEvent:
            fmt.Printf("We have been mentioned in %v", ev.Channel)
            _, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText("Yes, mentioned, ", false))
            if err != nil {
                fmt.Printf("failed posting message: %v", err)
            }

        case *slackevents.MemberJoinedChannelEvent:
            fmt.Printf("user %q joined to channel %q", ev.User, ev.Channel)
        }

    default:
        client.Debugf("unsupported Events API event received")
    }
}


func middlewareAppMentionEvent(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("middlewareAppMentionEvent")
	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	fmt.Printf("EventMention received middlewareAppMentionEvent: %+v\n", eventsAPIEvent)

	client.Ack(*evt.Request)

	ev, ok := eventsAPIEvent.InnerEvent.Data.(*slackevents.AppMentionEvent)
	if !ok {
		fmt.Printf("Ignored %+v\n", ev)
		return
	}

	fmt.Printf("We have been mentioned in %v\n", ev.Channel)
	_, _, err := client.Client.PostMessage(ev.Channel, slack.MsgOptionText("Oh, hello.", false))
	if err != nil {
		fmt.Printf("failed posting message: %v", err)
	}
}

func middlewareInteractive(evt *socketmode.Event, client *socketmode.Client) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	fmt.Printf("Interaction received: %+v\n", callback)

	var payload interface{}

	switch callback.Type {
	case slack.InteractionTypeBlockActions:
		// See https://api.slack.com/apis/connections/socket-implement#button
		client.Debugf("button clicked!")
	case slack.InteractionTypeShortcut:
	case slack.InteractionTypeViewSubmission:
		// See https://api.slack.com/apis/connections/socket-implement#modal
	case slack.InteractionTypeDialogSubmission:
	default:

	}

	client.Ack(*evt.Request, payload)
}

func middlewareInteractionTypeBlockActions(evt *socketmode.Event, client *socketmode.Client) {
	client.Debugf("button clicked!")
}

func middlewareSlashCommand(evt *socketmode.Event, client *socketmode.Client) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	client.Debugf("Slash command received: %+v", cmd)

    switch cmd.Command {
    case "/dadjoke":
        handleDadJokeCommand(evt, client)
    case "/weather":
        handleWeatherCommand(evt, client)
    case "/openai":
        handleOpenAICommand(evt, client)
    default:
        // If the command is not one of the specified commands, ignore and return
        fmt.Printf("Ignored %+v\n", evt)
        return
    }

}


func handleDadJokeCommand(evt *socketmode.Event, client *socketmode.Client) {
    client.Debugf("Slash command '/dadjoke' received: %+v", evt)

    // Add your response logic for the "/dadjoke" command here
    // Example response with a Dad joke
    //responseText := "Why don't scientists trust atoms? Because they make up everything! 😄"

    jokeText, err := getDadJoke()
    if err != nil {
        jokeText = "Not a Joke! " + err.Error()
    }

    payload := map[string]interface{}{
        "blocks": []slack.Block{
            slack.NewSectionBlock(
                &slack.TextBlockObject{
                    Type: slack.MarkdownType,
                    Text: jokeText, //foo
                },
                nil,
                slack.NewAccessory(
                    slack.NewButtonBlockElement(
                        "",
                        "somevalue",
                        &slack.TextBlockObject{
                            Type: slack.PlainTextType,
                            Text: "bar",
                        },
                    ),
                ),
            ),
        },
    }

    client.Ack(*evt.Request, payload)
}

type JokeResponse struct {
	Joke string `json:"joke"`
}

func getDadJoke() (string, error) {
	url := "https://icanhazdadjoke.com/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var jokeResp JokeResponse
	err = json.Unmarshal(body, &jokeResp)
	if err != nil {
		return "", err
	}

	return jokeResp.Joke, nil
}


func handleOpenAICommand(evt *socketmode.Event, client *socketmode.Client) {
    client.Debugf("Slash command '/openai' received: %+v", evt)

    // Add your response logic for the "/openai" command here

    inputTxt := evt.Data.(slack.SlashCommand).Text

    openaiResponse, openaiErr := getOpenAIResponse(inputTxt)
    if openaiErr != nil {
        openaiResponse = "ResponseError: " + openaiErr.Error()
    }

    payload := map[string]interface{}{
        "blocks": []slack.Block{
            slack.NewSectionBlock(
                &slack.TextBlockObject{
                    Type: slack.MarkdownType,
                    Text: openaiResponse,
                },
                nil,
                slack.NewAccessory(
                    slack.NewButtonBlockElement(
                        "",
                        "somevalue",
                        &slack.TextBlockObject{
                            Type: slack.PlainTextType,
                            Text: "openai",
                        },
                    ),
                ),
            ),
        },
    }

    client.Ack(*evt.Request, payload)
}


func getOpenAIResponse(prompt string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}



func handleWeatherCommand(evt *socketmode.Event, client *socketmode.Client) {
    client.Debugf("Slash command '/weather' received: %+v", evt)

    // Add your response logic for the "/weather" command here

    // Example response 
    responseText := "102 °F Temperatures are on the up!  the water is warm."

    payload := map[string]interface{}{
        "blocks": []slack.Block{
            slack.NewSectionBlock(
                &slack.TextBlockObject{
                    Type: slack.MarkdownType,
                    Text: responseText,
                },
                nil,
                slack.NewAccessory(
                    slack.NewButtonBlockElement(
                        "",
                        "somevalue",
                        &slack.TextBlockObject{
                            Type: slack.PlainTextType,
                            Text: "wet bar",
                        },
                    ),
                ),
            ),
        },
    }

    client.Ack(*evt.Request, payload)
}



func middlewareDefault(evt *socketmode.Event, client *socketmode.Client) {
	// fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
}


