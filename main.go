package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token   string
	GuildID string
	HostID  string
)

var q []*discordgo.User

func init() {

	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&GuildID, "g", "", "Guild in which voice channel exists")
	flag.StringVar(&HostID, "h", "", "Host of the mob session")
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
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers

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

func remove(slice []*discordgo.User, pos int) (newslice []*discordgo.User) {
	newslice = append(slice[:pos], slice[pos+1:]...)
	return
}

func idToMention(id string) (mention string) {
	mention = "<@" + id + ">"
	return
}

func qMsg(s *discordgo.User, w []*discordgo.User) (msg *discordgo.MessageEmbed) {
	var wm []string
	if len(w) == 0 {
		wm = append(wm, "No one yet!")
	} else {
		for i, user := range w {
			pos := strconv.Itoa(i + 1)
			wm = append(wm, pos+". "+idToMention(user.ID))
		}
	}

	msg = &discordgo.MessageEmbed{
		Title: "Queue:",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "Now Speaking:",
				Value:  idToMention(s.ID),
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
	guildMembers, err := s.GuildMembers(GuildID, "0", 1000)
	if err != nil {
		fmt.Println(err)
	}
	if len(args) == 0 {
		if len(q) == 0 {
			e := &discordgo.MessageEmbed{
				Title:       "No one in the queue yet!",
				Description: "Type `.q add` to add yourself to the queue.",
			}
			s.ChannelMessageSendEmbed(channel, e)
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
		q = append(q, msg.Author)

		if len(q) == 1 {
			for _, member := range guildMembers {
				fmt.Println("did .q add")
				err := s.GuildMemberMute(GuildID, member.User.ID, true)
				if err != nil {
					fmt.Println(err)
				}
				//time.Sleep(1 * time.Second)
			}
			s.GuildMemberMute(GuildID, msg.Author.ID, false)
		}

		e := &discordgo.MessageEmbed{
			Title:       "Added!",
			Description: idToMention(msg.Author.ID) + " has been added to the queue.",
		}
		s.ChannelMessageSendEmbed(channel, e)
		s.ChannelMessageSendEmbed(channel, qMsg(q[0], q[1:]))
	case "next", "pop":
		if msg.Author.ID == HostID || msg.Author == q[0] {
			fmt.Printf("%v", len(q))
			if len(q) == 1 {
				q = q[1:]
				for _, member := range guildMembers {
					s.GuildMemberMute(GuildID, member.User.ID, false)
				}
				e := &discordgo.MessageEmbed{
					Title:       "No one else in queue!",
					Description: "Unmuted everyone. Type `.q add` to restart the queue.",
				}
				s.ChannelMessageSendEmbed(channel, e)
				return
			} else if len(q) == 0 {
				e := &discordgo.MessageEmbed{
					Title:       "No one in queue!",
					Description: "Type `.q add` to add yourself to the queue.",
				}
				s.ChannelMessageSendEmbed(channel, e)
				return
			}
			s.GuildMemberMute(GuildID, q[0].ID, true)
			q = q[1:]
			s.GuildMemberMute(GuildID, q[0].ID, false)

			e := &discordgo.MessageEmbed{
				Title:       "Popped the stack!",
				Description: idToMention(q[0].ID) + " is now the speaker.",
			}
			s.ChannelMessageSendEmbed(channel, e)
			s.ChannelMessageSendEmbed(channel, qMsg(q[0], q[1:]))
		} else {
			e := &discordgo.MessageEmbed{
				Title:       "Oops, that didn't work.",
				Description: "Only the current speaker (" + idToMention(q[0].ID) + ") or the session host(" + idToMention(HostID) + ") can run `.q next`.",
			}
			s.ChannelMessageSendEmbed(channel, e)
		}
	case "remove", "rm":
		positionInt, err := strconv.Atoi(queueArgs[0])
		if msg.Author.ID == HostID || msg.Author == q[positionInt] {
			if err != nil {
				e := &discordgo.MessageEmbed{
					Title:       "Oops, that didn't work.",
					Description: "Please enter a valid integer position.",
				}
				s.ChannelMessageSendEmbed(channel, e)
			} else {
				removed := q[positionInt]
				q = remove(q, positionInt)
				e := &discordgo.MessageEmbed{
					Title:       "Removed " + removed.Username + " from the queue!",
					Description: idToMention(q[positionInt].ID) + " is now at position " + queueArgs[0],
				}
				s.ChannelMessageSendEmbed(channel, e)
				s.ChannelMessageSendEmbed(channel, qMsg(q[0], q[1:]))
			}
		} else {
			e := &discordgo.MessageEmbed{
				Title:       "Oops, that didn't work.",
				Description: "Only the session host (" + idToMention(HostID) + ") or the person to be removed can run `.q remove`.",
			}
			s.ChannelMessageSendEmbed(channel, e)
		}
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
