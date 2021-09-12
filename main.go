package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token string
)
var q []string

func init() {

	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func idToMention(id string) (mention string) {
	mention = "<@" + id + ">"
	return
}

func qMsg(s string, w []string) (msg *discordgo.MessageEmbed) {
	var wm []string
	for _, id := range w {
		wm = append(wm, idToMention(id))
	}

	msg = &discordgo.MessageEmbed{
		Title: "Queue:",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "Now Speaking:",
				Value:  idToMention(s),
				Inline: false,
			},
			&discordgo.MessageEmbedField{
				Name:   "Waiting:",
				Value:  strings.Join(wm, "\n"),
				Inline: false,
			},
		},
	}

	return
}

func queue(args []string, msg *discordgo.MessageCreate, channel string, s *discordgo.Session) {
	if len(args) == 0 {
		if len(q) == 0 {
			s.ChannelMessageSend(channel, "Nothing in the queue yet! Type `.q add` to add yourself to the queue.")
			return
		} else {
			s.ChannelMessageSendEmbed(channel, qMsg(q[0], q[1:]))
			return
		}
	}
	queueCmd := args[0]
	queueArgs := args[1:]
	switch queueCmd {
	case "add", "+":
		if len(queueArgs) == 0 {
			q = append(q, msg.Author.ID)
		}
		e := &discordgo.MessageEmbed{
			Title:       "Added!",
			Description: idToMention(msg.Author.ID) + " has been added to the waiting list.",
		}
		s.ChannelMessageSendEmbed(channel, e)
		s.ChannelMessageSendEmbed(channel, qMsg(q[0], q[1:]))
	case "next", "pop":
		q = q[1:]
		e := &discordgo.MessageEmbed{
			Title:       "Popped the stack!",
			Description: idToMention(q[0]) + " is now the speaker.",
		}
		s.ChannelMessageSendEmbed(channel, e)
		s.ChannelMessageSendEmbed(channel, qMsg(q[0], q[1:]))
	default:
		s.ChannelMessageSend(channel, "Type `.help` for a list of valid commands.")
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	prefix := "."

	// ignore messages not starting with prefix
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	message := strings.Split(m.Content, " ")
	fmt.Println(message)

	args := message[1:]
	fmt.Println(args)

	command := message[0]
	if len(command) == 1 {
		return
	}
	command = command[1:]
	fmt.Println(command)

	switch command {
	case "ping":
		// If the message is "ping" reply with "Pong!"
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "pong":
		// If the message is "pong" reply with "Ping!"
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	case "say":
		words := strings.Join(args, " ")
		s.ChannelMessageSend(m.ChannelID, words)
	case "timer":
		s.ChannelMessageSend(m.ChannelID, "`to be implemented`")
	case "q", "queue":
		queue(args, m, m.ChannelID, s)
	case "help":
		s.ChannelMessageSend(m.ChannelID, "`to be implemented`")
	default:
		s.ChannelMessageSend(m.ChannelID, "Type `.help` for a list of valid commands.")
	}
}
