package handler

import (
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

var _ bot.EventListener = (*Handler)(nil)

func New(logger log.Logger) *Handler {
	return &Handler{
		Logger:     logger,
		Commands:   map[string]Command{},
		Components: map[string]Component{},
		Modals:     map[string]Modal{},
		Message:    map[uuid.UUID]Message{},
		Ready:      []func(*events.Ready){},
	}
}

type Handler struct {
	Logger log.Logger

	Commands   map[string]Command
	Components map[string]Component
	Modals     map[string]Modal
	Message    map[uuid.UUID]Message
	Ready      []func(*events.Ready)
}

func (h *Handler) AddCommands(commands ...Command) {
	for _, command := range commands {
		h.Commands[command.Create.CommandName()] = command
	}
}

func (h *Handler) AddComponents(components ...Component) {
	for _, component := range components {
		h.Components[component.Name] = component
	}
}

func (h *Handler) AddComponent(component Component) func() {
	h.Components[component.Name] = component
	return func() {
		delete(h.Components, component.Name)
	}
}

func (h *Handler) AddModals(modals ...Modal) {
	for _, modal := range modals {
		h.Modals[modal.Name] = modal
	}
}

func (h *Handler) AddMessage(message Message) func() {
	h.Message[message.UUID] = message
	return func() {
		delete(h.Message, message.UUID)
	}
}

func (h *Handler) AddReady(ready func(*events.Ready)) {
	h.Ready = append(h.Ready, ready)
}

func (h *Handler) handleReady(e *events.Ready) {
	for _, v := range h.Ready {
		v(e)
	}
}

func (h *Handler) SyncCommands(client bot.Client, guildIDs ...snowflake.ID) {
	commands := make([]discord.ApplicationCommandCreate, len(h.Commands))
	var i int
	for _, command := range h.Commands {
		commands[i] = command.Create
		i++
	}

	if len(guildIDs) == 0 {
		if _, err := client.Rest().SetGlobalCommands(client.ApplicationID(), commands); err != nil {
			h.Logger.Error("Failed to sync global commands: ", err)
			return
		}
		h.Logger.Infof("Synced %d global commands", len(commands))
		return
	}

	for _, guildID := range guildIDs {
		if _, err := client.Rest().SetGuildCommands(client.ApplicationID(), guildID, commands); err != nil {
			h.Logger.Errorf("Failed to sync commands for guild %d: %s", guildID, err)
			continue
		}
		h.Logger.Infof("Synced %d commands for guild %s", len(commands), guildID)
	}
}

func (h *Handler) OnEvent(event bot.Event) {
	switch e := event.(type) {
	case *events.ApplicationCommandInteractionCreate:
		go h.handleCommand(e)
	case *events.AutocompleteInteractionCreate:
		go h.handleAutocomplete(e)
	case *events.ComponentInteractionCreate:
		go h.handleComponent(e)
	case *events.ModalSubmitInteractionCreate:
		go h.handleModal(e)
	case *events.MessageCreate:
		go h.handleMessage(e)
	case *events.Ready:
		go h.handleReady(e)
	}
}
