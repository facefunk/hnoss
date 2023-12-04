package hnoss

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

type DiscordChatAdapter struct {
	token           string
	defaultChanName string

	defaultChanID string
	c             chan string
	session       *discordgo.Session
	wg            sync.WaitGroup
}

func NewDiscordChatAdapter(token, defaultChanName string) *DiscordChatAdapter {
	d := &DiscordChatAdapter{
		token:           token,
		defaultChanName: defaultChanName,
		c:               make(chan string),
	}
	// New never actually returns an error
	d.session, _ = discordgo.New("Bot " + d.token)
	d.session.AddHandler(d.messageCreate)
	d.session.AddHandler(d.ready)
	return d
}

func (d *DiscordChatAdapter) Chan() <-chan string {
	return d.c
}

func (d *DiscordChatAdapter) Listen() error {
	err := d.session.Open()
	switch err {
	case discordgo.ErrWSAlreadyOpen:
		return nil
	case nil:
		// wait for Discord ready event before returning.
		d.wg.Add(1)
		d.wg.Wait()
		return nil
	default:
		return ErrorWrap(err, "failed to open Discord session")
	}
}

// Handler for Discord ready event.
func (d *DiscordChatAdapter) ready(s *discordgo.Session, r *discordgo.Ready) {
	for _, guild := range r.Guilds {
		chans, _ := s.GuildChannels(guild.ID)
		for _, c := range chans {
			if c.Type == discordgo.ChannelTypeGuildText && c.Name == d.defaultChanName {
				d.defaultChanID = c.ID
			}
		}
	}
	d.wg.Done()
}

// Handler called when anyone creates a message in a Guild that the bot is a member of.
func (d *DiscordChatAdapter) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Answer if mentioned
	for _, mention := range m.Mentions {
		if mention.ID == s.State.User.ID {
			d.c <- m.ChannelID
		}
	}
}

func (d *DiscordChatAdapter) Close() error {
	if err := d.session.Close(); err != nil {
		return ErrorWrap(err, "failed to close Discord session")
	}
	return nil
}

func (d *DiscordChatAdapter) Post(chanID, msg string) error {
	if chanID == "" {
		chanID = d.defaultChanID
	}
	_, err := d.session.ChannelMessageSend(chanID, msg)
	if err != nil {
		return ErrorWrap(err, "failed to send Discord message")
	}
	return nil
}
