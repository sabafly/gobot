package components

import (
	"log/slog"
	"strings"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
)

var (
	DefaultCommands []Command
)

func (c *Components) AddCommands(cmds ...Command) {
	for _, cmd := range cmds {
		c.commandsRegistry[cmd.Name()] = cmd
	}
}

func (c *Components) AddCommand(cmd Command) {
	c.commandsRegistry[cmd.Name()] = cmd
}

func (c *Components) Initialize(client bot.Client) error {
	for _, cmd := range DefaultCommands {
		c.AddCommand(cmd)
	}

	var commands, privCommands []discord.ApplicationCommandCreate
	for name, cmd := range c.commandsRegistry {
		if len(cmd.Scheduler()) > 0 {
			slog.Info("コマンドスケジューラ―を登録します", "name", name, "count", len(cmd.Create()))
			for _, s := range cmd.Scheduler() {
				go execSchedule(c, client, s)
			}
		}
		if len(cmd.Create()) > 0 {
			slog.Info("コマンドを登録します", "name", name, "count", len(cmd.Create()), "is_private", cmd.IsPrivate())
			if cmd.IsPrivate() {
				privCommands = append(privCommands, cmd.Create()...)
			} else {
				commands = append(commands, cmd.Create()...)
			}
		}
	}

	if _, err := client.Rest().SetGlobalCommands(client.ApplicationID(), commands); err != nil {
		slog.Error("コマンドの登録に失敗", slog.Any("err", err), slog.String("body", string(err.(rest.Error).RsBody)))
		return err
	}

	for _, id := range c.Config().Debug.DebugGuilds {
		if _, err := client.Rest().SetGuildCommands(client.ApplicationID(), id, privCommands); err != nil {
			slog.Error("プライベートコマンドの登録に失敗", "err", err, "guild", id)
			return err
		}
	}

	client.EventManager().AddEventListeners(
		bot.NewListenerFunc(c.OnEvent()),
		&events.ListenerAdapter{
			OnGuildJoin:  c.OnGuildJoin(),
			OnGuildLeave: c.OnGuildLeave(),
		},
	)
	return nil
}

func (c *Components) OnEvent() func(bot bot.Event) {
	return func(event bot.Event) {
		switch e := event.(type) {
		case *events.ApplicationCommandInteractionCreate:
			cmd, ok := c.commandsRegistry[e.Data.CommandName()]
			if !ok {
				slog.Warn("不明なコマンド", "command_name", e.Data.CommandName())
				return
			}
			h := cmd.CommandHandler()
			if h == nil {
				slog.Warn("コマンド処理がnil", "custom_id", e.Data.CommandName())
				return
			}
			if err := h(e); err != nil {
				slog.Error("コマンド処理中にエラーが発生しました", "err", err)
			}
		case *events.ComponentInteractionCreate:
			namespace := strings.Split(e.Data.CustomID(), ":")
			cmd, ok := c.commandsRegistry[namespace[0]]
			if !ok {
				slog.Warn("不明なコンポーネント", "custom_id", e.Data.CustomID())
				return
			}
			h := cmd.ComponentHandler()
			if h == nil {
				slog.Warn("コンポーネント処理がnil", "custom_id", e.Data.CustomID())
				return
			}
			if err := h(e); err != nil {
				slog.Error("コンポーネント処理中にエラーが発生しました", "err", err)
				return
			}
		case *events.ModalSubmitInteractionCreate:
			namespace := strings.Split(e.Data.CustomID, ":")
			cmd, ok := c.commandsRegistry[namespace[0]]
			if !ok {
				slog.Warn("不明なモーダル提出インタラクション", "custom_id", e.Data.CustomID)
				return
			}
			h := cmd.ModalHandler()
			if h == nil {
				slog.Warn("モーダル提出インタラクション処理がnil", "custom_id", e.Data.CustomID)
				return
			}
			if err := h(e); err != nil {
				slog.Error("モーダル提出インタラクション処理中にエラーが発生しました", "err", err)
				return
			}
		case *events.AutocompleteInteractionCreate:
			namespace := strings.Split(e.Data.CommandName, ":")
			cmd, ok := c.commandsRegistry[namespace[0]]
			if !ok {
				slog.Warn("不明なオートコンプリート", "custom_id", e.Data.CommandName)
				return
			}
			h := cmd.AutocompleteHandler()
			if h == nil {
				slog.Warn("オートコンプリート処理がnil", "custom_id", e.Data.CommandName)
				return
			}
			if err := h(e); err != nil {
				slog.Error("オートコンプリート処理中にエラーが発生しました", "err", err)
				return
			}
		default:
			for name, cmd := range c.commandsRegistry {
				if h := cmd.OnEvent(); h != nil {
					if err := h(event); err != nil {
						slog.Error("イベント処理中にエラーが発生しました", "err", err, "cmd_name", name)
					}
				}
			}
		}
	}
}

type Command interface {
	Name() string

	Create() []discord.ApplicationCommandCreate
	IsPrivate() bool
	CommandHandler() func(event *events.ApplicationCommandInteractionCreate) error
	ComponentHandler() func(event *events.ComponentInteractionCreate) error
	ModalHandler() func(event *events.ModalSubmitInteractionCreate) error
	AutocompleteHandler() func(event *events.AutocompleteInteractionCreate) error
	OnEvent() func(event bot.Event) error
	Scheduler() []Scheduler
}
