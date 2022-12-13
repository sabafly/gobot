package session

import (
	"errors"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/ikafly144/gobot/pkg/types"
)

type SessionID struct {
	ID string
}

type Session struct {
	id   SessionID
	Data interface{}
	Type SessionType
}

func (s *Session) ID() SessionID {
	return s.id
}

type SessionType int

const (
	RolePanelEdit SessionType = 1
)

var sessionManager map[SessionID]*Session = make(map[SessionID]*Session)

func NewSession(id SessionID, t SessionType) (*Session, error) {

	if s, ok := sessionManager[id]; ok && s.Type == t {
		return s, nil
	}

	s := &Session{
		id:   id,
		Data: nil,
		Type: t,
	}
	sessionManager[id] = s

	return s, nil
}

func GetSession(id SessionID) (*Session, error) {

	if s, ok := sessionManager[id]; ok {
		return s, nil
	}

	return &Session{}, errors.New("no session found")
}

func RemoveSession(id SessionID) error {

	delete(sessionManager, id)

	return nil
}

var handler map[SessionType]func(s *discordgo.Session, m *discordgo.MessageCreate, session *Session) = make(map[SessionType]func(s *discordgo.Session, m *discordgo.MessageCreate, session *Session))

func AddHandler(s SessionType, h func(s *discordgo.Session, m *discordgo.MessageCreate, session *Session)) {
	handler[s] = h
}

func HandleExec(s *discordgo.Session, m *discordgo.MessageCreate) {
	d, ok := sessionManager[SessionID{ID: m.Author.ID}]
	if !ok {
		return
	}
	log.Print("session")
	log.Printf("%+v", d.Type)
	switch d.Type {
	case RolePanelEdit:
		data, _ := d.Data.(types.PanelEmojiConfig)
		if data.MessageData.ChannelID != m.ChannelID {
			log.Print("not same channel")
			return
		}
		h, ok := handler[RolePanelEdit]
		if !ok {
			log.Print("handler not found")
			return
		}
		h(s, m, d)
	}
}
