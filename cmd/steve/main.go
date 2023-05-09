package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davidlawson7/steve-gobot/pkg/bot"
	"github.com/davidlawson7/steve-gobot/pkg/constants"
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
	CurrentIP     string = ""
	Reachable     bool   = false
	Token         string
	Shutdown      bool = true
	ShutdownMutex sync.Mutex

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
			// Server started and reachable, no reason to hit lambda
			if CurrentIP != "" && Reachable {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Minecraft server already started. IP: %s", CurrentIP),
					},
				})
				return
			}

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

			_, _, err = bot.ParseLambda(res.Payload)
			if err != nil {
				// Handle better
				fmt.Println(err)
				os.Exit(1)
			}

			ShutdownMutex.Lock()
			Shutdown = false
			ShutdownMutex.Unlock()

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

					if body.Message == "running" {
						if _, err := minequery.Ping17(body.IPAddress, 25565); err == nil {
							// Only if the EC2 is running AND MC server started and reachable do we proceed with ending this loop
							Reachable = true
						}
					}

					if body.Message == "running" && Reachable {
						CurrentIP = body.IPAddress // Set CURRENT IP address
						message := fmt.Sprintf("Started Minecraft, IP: %s\n", CurrentIP)
						s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
							Content: &message,
						})

						// Create a scheduled job to check the server if empty after 10 minutes, and then shutdown
						go func() {
							// Initial sleep to prevent spamming the server as soon as its ready
							time.Sleep(constants.TIME_5_MIN)
							mark := false

							for {
								mping, err := minequery.Ping17(CurrentIP, 25565)
								if err != nil {
									// Cannot reach server, it must be down
									fmt.Println("mcserver not reachable during normal ping, must be down.")
									s.ChannelMessageSend(i.ChannelID, "mcserver not reachable during normal ping, must be down... Help sir.")
									break
								}
								isEmpty := len(mping.SamplePlayers) == 0

								// triggered a countdown to turn server off, but players jumped back on
								if mark && !isEmpty {
									mark = false
								}

								// SHUTDOWN SERVER
								if mark && isEmpty {
									ShutdownMutex.Lock()
									if Shutdown {
										fmt.Println("already shutdown manually, this thread can just die silently")
										break // If shutdown already, just exit the loop and end thread
									}
									ShutdownMutex.Unlock()

									fmt.Println("Max time reached without players in server, shut it down.")
									if _, err := bot.InvokeLambda(&ctx, "stop"); err != nil {
										s.ChannelMessageSend(i.ChannelID, "Server left unattended for over 10 minutes BUT unable to auto shutdown... Help sir.")
									} else {
										s.ChannelMessageSend(i.ChannelID, "Server left unattended for over 10 minutes, shutting down...")
										// Reset control vars
										Reachable = false
										CurrentIP = ""
									}

									break
								}

								if isEmpty {
									mark = true
								}

								// Normal check interval, sleep for 10 minutes between doing work
								time.Sleep(constants.TIME_10_MIN)
							}

							// If we get here, thread should end
						}()

						// Server started! Send the response back to server
						return
					}
					counter++
					time.Sleep(constants.TIME_10_SEC)
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

			_, _, err = bot.ParseLambda(res.Payload)
			if err != nil {
				// Handle better
				fmt.Println(err)
				os.Exit(1)
			}

			ShutdownMutex.Lock()
			Shutdown = true
			ShutdownMutex.Unlock()

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

					if body.Message == "stopped" {
						CurrentIP = ""    // Unset CURRENT IP address
						Reachable = false // Set status to unreachable
						message := "Stopped Minecraft"
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
			if CurrentIP == "" {
				// EC2 is not running, or atleast was started outside this apps flow
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Server is off.",
					},
				})
				// Exit early
				return
			}

			// Send message to indicate PING is happening.
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Awaiting ping response...",
				},
			})

			res, err := minequery.Ping17(CurrentIP, 25565)

			if err != nil {
				errorContent := "Server is on but Minecraft is not yet reachable."
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &errorContent,
				})
				return
			}

			var players = ""
			for _, p := range res.SamplePlayers {
				if players == "" {
					players = p.Nickname
					continue
				}
				players = fmt.Sprintf("%s, %s", players, p.Nickname)
			}
			resContent := fmt.Sprintf("Minecraft Server %s, protocol version %d\ndescription: %s\n%d/%d players online\nplayers: %s\n", res.VersionName, res.ProtocolVersion, bot.NaturalizeMOTD(res.Description.String()), res.OnlinePlayers, res.MaxPlayers, players)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &resContent,
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
	// In this example, we only care about receiving message events.
	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentGuildPresences

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
	for _, v := range registeredCommands {
		err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	log.Println("Gracefully shutting down.")
}
