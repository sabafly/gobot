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
		Logger:      logger,
		Commands:    map[string]Command{},
		Components:  map[string]Component{},
		Modals:      map[string]Modal{},
		Message:     map[uuid.UUID]Message{},
		Ready:       []func(*events.Ready){},
		MemberJoin:  map[uuid.UUID]MemberJoin{},
		MemberLeave: map[uuid.UUID]MemberLeave{},

		ExcludeID: map[snowflake.ID]struct{}{},
	}
}

type Handler struct {
	Logger log.Logger

	Commands    map[string]Command
	Components  map[string]Component
	Modals      map[string]Modal
	Message     map[uuid.UUID]Message
	Ready       []func(*events.Ready)
	MemberJoin  map[uuid.UUID]MemberJoin
	MemberLeave map[uuid.UUID]MemberLeave

	ExcludeID  map[snowflake.ID]struct{}
	DevGuildID []snowflake.ID
}

func (h *Handler) AddExclude(ids ...snowflake.ID) {
	for _, id := range ids {
		h.ExcludeID[id] = struct{}{}
	}
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

func (h *Handler) AddMemberJoin(memberJoin MemberJoin) func() {
	h.MemberJoin[memberJoin.UUID] = memberJoin
	return func() {
		delete(h.MemberJoin, memberJoin.UUID)
	}
}

func (h *Handler) AddMemberJoins(memberJoins ...MemberJoin) {
	for _, mj := range memberJoins {
		h.MemberJoin[mj.UUID] = mj
	}
}

func (h *Handler) AddMemberLeave(memberLeave MemberLeave) func() {
	h.MemberLeave[memberLeave.UUID] = memberLeave
	return func() {
		delete(h.MemberLeave, memberLeave.UUID)
	}
}

func (h *Handler) AddMemberLeaves(memberLeaves ...MemberLeave) {
	for _, ml := range memberLeaves {
		h.MemberLeave[ml.UUID] = ml
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
	commands := []discord.ApplicationCommandCreate{}
	devCommands := []discord.ApplicationCommandCreate{}
	for _, command := range h.Commands {
		if command.DevOnly {
			devCommands = append(devCommands, command.Create)
		} else {
			commands = append(commands, command.Create)
		}
	}

	if len(devCommands) > 0 {
		for _, id := range h.DevGuildID {
			if _, err := client.Rest().SetGuildCommands(client.ApplicationID(), id, devCommands); err != nil {
				h.Logger.Errorf("Failed to sync %d commands: %s", id, err)
			}
			h.Logger.Infof("Synced %d guild %d commands", len(devCommands), id)
			cmd, err := client.Rest().GetGuildCommands(client.ApplicationID(), id, true)
			h.Logger.Debugf("%+v %s", *cmd[0].GuildID(), err)
		}
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
	go func() {
		defer h.panicCatch()
		switch e := event.(type) {
		case *events.ApplicationCommandInteractionCreate:
			h.handleCommand(e)
		case *events.AutocompleteInteractionCreate:
			h.handleAutocomplete(e)
		case *events.ComponentInteractionCreate:
			h.handleComponent(e)
		case *events.ModalSubmitInteractionCreate:
			h.handleModal(e)
		case *events.MessageCreate:
			h.handleMessage(e)
		case *events.Ready:
			h.handleReady(e)
		case *events.GuildMemberJoin:
			h.handlerMemberJoin(e)
		case *events.GuildMemberLeave:
			h.handlerMemberLeave(e)
		}
	}()
}

func (h *Handler) panicCatch() {
	if err := recover(); err != nil {
		h.Logger.Errorf("panic: %s", err)
	}
}
