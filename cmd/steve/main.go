package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davidlawson7/steve-gobot/pkg/bot"
	"github.com/dreamscached/minequery/v2"
)

type Request struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type MCStartResponse struct {
	Status int                 `json:"statusCode,omitempty"`
	Body   MCStartResponseBody `json:"body,omitempty"`
}

type MCStartResponseBody struct {
	Message   string `json:"message"`
	IPAddress string `json:"ipAddress,omitempty"`
}

// const KuteGoAPIURL = "https://kutego-api-xxxxx-ew.a.run.app"
var s *discordgo.Session

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func init() {
	var err error
	s, err = discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

var (
	integerOptionMinValue = 1.0
	CurrentIP             string
	Token                 string

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "start",
			Description: "Sends a start message to boot up the Minecraft Server.",
		},
		{
			Name:        "stop",
			Description: "Sends a stops message to shutdown the Minecraft Server.",
		},
		{
			Name:        "status",
			Description: "Requests information about the current servers state.",
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"start": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Send response immediately and handle the starting logic later.
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Starting Minecraft...",
				},
			})

			ctx := context.Background()

			res, err := bot.InvokeLambda(&ctx, "start")
			if err != nil {
				// Handle better
				fmt.Println(err)
				os.Exit(1)
			}

			_, body, err := bot.ParseLambda(res.Payload)
			if err != nil {
				// Handle better
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Println("Initial Start Response: ", body.Message)

			go func() {
				counter := 0

				for counter <= 20 {
					res, err := bot.InvokeLambda(&ctx, "start")
					if err != nil {
						// Handle better
						fmt.Println(err)
						os.Exit(1)
					}

					_, body, err := bot.ParseLambda(res.Payload)
					if err != nil {
						// Handle better
						fmt.Println(err)
						os.Exit(1)
					}

					fmt.Println("Check Response: ", body.Message)

					if body.Message == "running" {
						CurrentIP = body.IPAddress // Set CURRENT IP address
						message := fmt.Sprintf("Started Minecraft, IP: %s\n", body.IPAddress)
						s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
							Content: &message,
						})
						// Server started! Send the response back to server
						return
					}
					counter++
					time.Sleep(time.Duration(10) * time.Second)
				}

				// Couldnt start the server, send a update response to the channel
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Failed to start Minecraft after %d checks.\n", counter),
					},
				})
			}() // Note the parentheses. We must call the anonymous function.

		},
		"stop": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Send response immediately and handle the starting logic later.
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Stopping Minecraft...",
				},
			})

			ctx := context.Background()

			res, err := bot.InvokeLambda(&ctx, "stop")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			_, body, err := bot.ParseLambda(res.Payload)
			if err != nil {
				// Handle better
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Println("Initial Stop Response: ", body.Message)

			go func() {
				counter := 0

				for counter <= 20 {
					res, err := bot.InvokeLambda(&ctx, "stop")
					if err != nil {
						// Handle better
						fmt.Println(err)
						os.Exit(1)
					}

					_, body, err := bot.ParseLambda(res.Payload)
					if err != nil {
						// Handle better
						fmt.Println(err)
						os.Exit(1)
					}

					fmt.Println("Check Response: ", body.Message)

					if body.Message == "stopped" {
						CurrentIP = "" // Unset CURRENT IP address
						message := fmt.Sprintf("Stopped Minecraft\n")
						s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
							Content: &message,
						})
						// Server started! Send the response back to server
						return
					}
					counter++
					time.Sleep(time.Duration(10) * time.Second)
				}

				// Couldnt start the server, send a update response to the channel
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Failed to stop Minecraft after %d checks.\n", counter),
					},
				})
			}() // Note the parentheses. We must call the anonymous function.
		},
		"status": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			fmt.Println("Do work status here")

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Status!",
				},
			})
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	s.AddHandler(messageCreate)
	// Register the messageCreate func as a callback for MessageCreate events.

	// In this example, we only care about receiving message events.
	s.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err := s.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))

	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")

	<-stop

	log.Println("Removing commands...")
	// // We need to fetch the commands, since deleting requires the command ID.
	// // We are doing this from the returned commands on line 375, because using
	// // this will delete all the commands, which might not be desirable, so we
	// // are deleting only the commands that we added.
	// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
	// if err != nil {
	// 	log.Fatalf("Could not fetch registered commands: %v", err)
	// }

	for _, v := range registeredCommands {
		err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	log.Println("Gracefully shutting down.")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "mcserver status" {

		if CurrentIP == "" {
			s.ChannelMessageSend(m.ChannelID, "Server is currently offline")
			return
		}

		res, err := minequery.Ping17(CurrentIP, 25565)

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to ping server")
			return
		}
		fmt.Println(res)
		s.ChannelMessageSend(m.ChannelID, res.String())
	}
}
