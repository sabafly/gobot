package game

import (
	"cmp"
	"fmt"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/bot/components/generic"
	"github.com/sabafly/gobot/ent"
	"github.com/sabafly/gobot/ent/chinchiroplayer"
	"github.com/sabafly/gobot/ent/chinchirosession"
	"github.com/sabafly/gobot/internal/builtin"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/translate"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
)

func Command(c *components.Components) *generic.Command {
	return (&generic.Command{
		Namespace: "game",
		Private:   true,
		CommandCreate: []discord.ApplicationCommandCreate{
			discord.SlashCommandCreate{
				Name:        "game",
				Description: "game command",
				IntegrationTypes: []discord.ApplicationIntegrationType{
					discord.ApplicationIntegrationTypeGuildInstall,
				},
				Contexts: []discord.InteractionContextType{
					discord.InteractionContextTypeGuild,
				},
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionSubCommand{
						Name:        "chinchirorin",
						Description: "chinchirorin command",
					},
				},
			},
		},
		CommandHandlers: map[string]generic.PermissionCommandHandler{
			"/game/chinchirorin": generic.PCommandHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("game.chinchirorin"),
				},
				CommandHandler: func(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
					if err := event.CreateMessage(discord.NewMessageBuilder().
						SetEmbeds(
							discord.NewEmbedBuilder().
								SetTitle(translate.Message(event.Locale(), "component.game.cinchiro.start.embed.title")).
								SetDescription(translate.Message(event.Locale(), "component.game.cinchiro.start.embed.description")).
								Build(),
						).
						SetContainerComponents(
							discord.NewActionRow(
								discord.ButtonComponent{
									Label:    translate.Message(event.Locale(), "component.game.cinchiro.start.button.label"),
									Style:    discord.ButtonStylePrimary,
									CustomID: "game:chinchirorin-create",
								},
							),
						).
						SetFlags(discord.MessageFlagEphemeral).
						BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
		},
		ComponentHandlers: map[string]generic.PermissionComponentHandler{
			"game:chinchirorin-create": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("game.chinchirorin.create"),
					generic.PermissionDefaultString("game.chinchirorin.play"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					// 作成ボタンを消す
					if err := event.UpdateMessage(discord.NewMessageBuilder().ClearContainerComponents().BuildUpdate()); err != nil {
						return errors.NewError(err)
					}
					// ギルドを取得
					guild, err := c.GuildCreateID(event, *event.GuildID())
					if err != nil {
						return errors.NewError(err)
					}
					// セッションを作成
					session, err := c.DB().ChinchiroSession.Create().
						SetGuild(guild).
						Save(event)
					if err != nil {
						return errors.NewError(err)
					}
					// セッションにプレイヤー(親)を追加
					if err := c.DB().ChinchiroPlayer.Create().SetIsOwner(true).SetUserID(event.User().ID).SetSession(session).Exec(event); err != nil {
						return errors.NewError(err)
					}
					// メッセージを作成
					if _, err := event.Client().Rest().CreateMessage(event.Channel().ID(), discord.NewMessageBuilder().
						SetEmbeds(
							discord.NewEmbedBuilder().
								SetTitle(translate.Message(event.Locale(), "component.game.cinchiro.created.embed.title")).
								SetDescription(translate.Message(event.Locale(), "component.game.cinchiro.created.embed.description")).
								SetFields(
									discord.EmbedField{
										Name:  translate.Message(event.Locale(), "component.game.cinchiro.created.embed.field.players.name"),
										Value: "- " + event.User().Mention(),
									},
								).
								Build(),
						).
						SetContainerComponents(
							discord.NewActionRow(
								discord.ButtonComponent{
									Label:    translate.Message(event.Locale(), "component.game.cinchiro.join.button.label"),
									Style:    discord.ButtonStylePrimary,
									CustomID: fmt.Sprintf("game:chinchirorin-join:%s", session.ID),
								},
							),
						).
						BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"game:chinchirorin-join": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("game.chinchirorin.play"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					id, err := uuid.Parse(args[2])
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}

					// セッション取得
					session, err := c.DB().ChinchiroSession.Query().Where(chinchirosession.ID(id)).First(event)
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}
					// 既に参加していたらエラー
					if session.QueryPlayers().Where(chinchiroplayer.UserID(event.User().ID)).ExistX(event) {
						return errors.NewError(errors.ErrorMessage("errors.already_exist", event))
					}
					// 参加
					if _, err := c.DB().ChinchiroPlayer.Create().SetUserID(event.User().ID).SetSession(session).Save(event); err != nil {
						return errors.NewError(err)
					}
					// プレイヤーを全員取得
					players := session.QueryPlayers().AllX(event)
					// プレイヤーをソート
					slices.SortStableFunc(players, func(a, b *ent.ChinchiroPlayer) int {
						return cmp.Compare(a.ID.Time(), b.ID.Time())
					})
					// プレイヤーを表示
					playerStr := ""
					for _, p := range players {
						playerStr += fmt.Sprintf("- %s\n", discord.UserMention(p.UserID))
					}
					// メッセージを更新
					if err := event.UpdateMessage(discord.NewMessageBuilder().
						SetEmbeds(
							discord.NewEmbedBuilder().
								SetTitle(translate.Message(event.Locale(), "component.game.cinchiro.created.embed.title")).
								SetDescription(translate.Message(event.Locale(), "component.game.cinchiro.created.embed.description")).
								SetFields(
									discord.EmbedField{
										Name:  translate.Message(event.Locale(), "component.game.cinchiro.created.embed.field.players.name"),
										Value: playerStr,
									},
								).
								Build(),
						).
						SetContainerComponents(
							discord.NewActionRow(
								discord.ButtonComponent{
									Label:    translate.Message(event.Locale(), "component.game.cinchiro.join.button.label"),
									Style:    discord.ButtonStylePrimary,
									CustomID: fmt.Sprintf("game:chinchirorin-join:%s", session.ID),
								},
								discord.ButtonComponent{
									Label:    translate.Message(event.Locale(), "component.game.cinchiro.start.button.label"),
									Style:    discord.ButtonStyleSuccess,
									CustomID: fmt.Sprintf("game:chinchirorin-start:%s", session.ID),
								},
							),
						).
						BuildUpdate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"game:chinchirorin-start": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("game.chinchirorin.start"),
					generic.PermissionDefaultString("game.chinchirorin.play"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					id, err := uuid.Parse(args[2])
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}

					// セッション取得
					session, err := c.DB().ChinchiroSession.Query().Where(chinchirosession.ID(id)).First(event)
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}
					// プレイヤーを全員取得
					players := session.QueryPlayers().AllX(event)
					if len(players) < 2 {
						return errors.NewError(errors.ErrorMessage("errors.not_enough.player", event))
					}
					// オーナーでないならエラー
					if !players[slices.IndexFunc(players, func(player *ent.ChinchiroPlayer) bool {
						return player.UserID == event.User().ID
					})].IsOwner {
						return errors.NewError(errors.ErrorMessage("errors.game.chinchiro.not_owner", event))
					}
					for i, player := range players {
						players[i] = player.Update().SetPoint(2000).SaveX(event)
					}
					// ターンを進める
					session = session.Update().AddTurn(1).SaveX(event)
					// 開始ボタンを消す
					_ = event.UpdateMessage(discord.NewMessageBuilder().ClearContainerComponents().BuildUpdate())
					// ゲーム開始
					if _, err := event.Client().Rest().CreateMessage(event.Channel().ID(), discord.NewMessageBuilder().
						SetEmbeds(
							chinchiroEmbed(event.Locale(), players),
						).
						SetContainerComponents(
							discord.NewActionRow(
								discord.ButtonComponent{
									Label:    translate.Message(event.Locale(), "component.game.cinchiro.game.bet.button.label"),
									Style:    discord.ButtonStylePrimary,
									CustomID: fmt.Sprintf("game:chinchirorin-bet:%s", session.ID),
								},
							),
						).
						BuildCreate(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"game:chinchirorin-bet": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("game.chinchirorin.play"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					id, err := uuid.Parse(args[2])
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}

					// セッション取得
					session, err := c.DB().ChinchiroSession.Query().Where(chinchirosession.ID(id)).First(event)
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}
					// プレイヤーを全員取得
					players := session.QueryPlayers().AllX(event)
					if len(players) < 2 {
						return errors.NewError(errors.ErrorMessage("errors.not_enough.player", event))
					}
					// プレイヤーをソート
					slices.SortStableFunc(players, func(a, b *ent.ChinchiroPlayer) int {
						return cmp.Compare(a.ID.Time(), b.ID.Time())
					})
					// 親は賭けられない
					if players[session.Turn-1%len(players)].UserID != event.User().ID {
						return errors.NewError(errors.ErrorMessage("errors.game.chinchiro.parent_cant_bet", event))
					}
					// 自分のプレイヤーを取得
					i := slices.IndexFunc(players, func(player *ent.ChinchiroPlayer) bool {
						return player.UserID == event.User().ID
					})
					// 参加していない場合はエラー
					if i == -1 {
						return errors.NewError(errors.ErrorMessage("errors.game.not_joined", event))
					}
					player := players[i]
					// 既に賭けている場合はエラー
					if player.Bet != nil {
						return errors.NewError(errors.ErrorMessage("errors.game.chinchiro.already_bet", event))
					}
					// 入力モーダルを表示
					if err := event.Modal(discord.NewModalCreateBuilder().
						SetTitle(translate.Message(event.Locale(), "component.game.cinchiro.game.bet.modal.title")).
						SetCustomID(fmt.Sprintf("game:chinchirorin-bet:%s", session.ID)).
						SetContainerComponents(
							discord.NewActionRow(
								discord.TextInputComponent{
									CustomID:  "value",
									Label:     translate.Message(event.Locale(), "component.game.cinchiro.game.bet.modal.input.label"),
									Style:     discord.TextInputStyleShort,
									MinLength: builtin.Ptr(1),
									Required:  true,
								},
							),
						).
						Build(),
					); err != nil {
						return errors.NewError(err)
					}
					return nil
				},
			},
			"game:chinchirorin-roll": generic.PComponentHandler{
				Permission: []generic.Permission{
					generic.PermissionDefaultString("game.chinchirorin.play"),
				},
				ComponentHandler: func(c *components.Components, event *events.ComponentInteractionCreate) errors.Error {
					args := strings.Split(event.Data.CustomID(), ":")
					id, err := uuid.Parse(args[2])
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}

					// セッション取得
					session, err := c.DB().ChinchiroSession.Query().Where(chinchirosession.ID(id)).First(event)
					if err != nil {
						return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
					}
					// プレイヤーを全員取得
					players := session.QueryPlayers().AllX(event)
					if len(players) < 2 {
						return errors.NewError(errors.ErrorMessage("errors.not_enough.player", event))
					}

					// サイコロを振る
					_ = []int{rand.N(6) + 1, rand.N(6) + 1, rand.N(6) + 1}

					// 親の番だった時
					if session.Loop == 0 {
						// 親じゃなかったらエラー
						if players[session.Turn-1%len(players)].UserID != event.User().ID {
							return errors.NewError(errors.ErrorMessage("errors.game.chinchiro.not_parent", event))
						}

					}
					return nil
				},
			},
		},
		ModalHandlers: map[string]generic.ModalHandler{
			"game:chinchirorin-bet": func(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
				args := strings.Split(event.Data.CustomID, ":")
				id, err := uuid.Parse(args[2])
				if err != nil {
					return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
				}

				// セッション取得
				session, err := c.DB().ChinchiroSession.Query().Where(chinchirosession.ID(id)).First(event)
				if err != nil {
					return errors.NewError(errors.ErrorMessage("errors.not_exist", event))
				}
				// プレイヤーを全員取得
				players := session.QueryPlayers().AllX(event)
				if len(players) < 2 {
					return errors.NewError(errors.ErrorMessage("errors.not_enough.player", event))
				}
				// プレイヤーをソート
				slices.SortStableFunc(players, func(a, b *ent.ChinchiroPlayer) int {
					return cmp.Compare(a.ID.Time(), b.ID.Time())
				})
				// 親は賭けられない
				if players[session.Turn-1%len(players)].UserID != event.User().ID {
					return errors.NewError(errors.ErrorMessage("errors.game.chinchiro.parent_cant_bet", event))
				}
				// 自分のプレイヤーを取得
				i := slices.IndexFunc(players, func(player *ent.ChinchiroPlayer) bool {
					return player.UserID == event.User().ID
				})
				// 参加していない場合はエラー
				if i == -1 {
					return errors.NewError(errors.ErrorMessage("errors.game.not_joined", event))
				}
				player := players[i]
				// 既に賭けている場合はエラー
				if player.Bet != nil {
					return errors.NewError(errors.ErrorMessage("errors.game.chinchiro.already_bet", event))
				}
				// 数値に変換
				value, err := strconv.Atoi(event.ModalSubmitInteraction.Data.Text("value"))
				if err != nil {
					return errors.NewError(errors.ErrorMessage("errors.game.chinchiro.invalid_bet", event))
				}
				// 1未満か所持ポイントより多い場合はエラー
				if value < 1 || player.Point < value {
					return errors.NewError(errors.ErrorMessage("errors.game.chinchiro.invalid_bet", event))
				}
				// 賭ける
				player, err = player.Update().AddPoint(-value).SetBet(value).Save(event)
				if err != nil {
					return errors.NewError(err)
				}
				players[i] = player
				// 賭けたことを通知
				if err := event.UpdateMessage(
					discord.NewMessageBuilder().
						SetEmbeds(
							chinchiroEmbed(event.Locale(), players),
						).
						BuildUpdate(),
				); err != nil {
					return errors.NewError(err)
				}
				if _, err := event.Client().Rest().CreateMessage(event.Channel().ID(), discord.NewMessageBuilder().
					SetEmbeds(
						discord.NewEmbedBuilder().
							SetDescription(translate.Message(event.Locale(), "component.game.cinchiro.game.bet.embed.description",
								translate.WithTemplate(map[string]any{
									"value":  value,
									"player": discord.UserMention(event.User().ID),
								}),
							)).
							Build(),
					).
					BuildCreate(),
				); err != nil {
					return errors.NewError(err)
				}

				// 全員掛けたら次のフェーズへ
				for i2, chinchiroPlayer := range players {
					if chinchiroPlayer.Bet == nil {
						return nil
					}
					if i2 == len(players)-1 {
						// メッセージを作成
						if _, err := event.Client().Rest().CreateMessage(event.Channel().ID(), discord.NewMessageBuilder().
							SetEmbeds(
								discord.NewEmbedBuilder().
									SetTitle(translate.Message(event.Locale(), "component.game.cinchiro.game.roll.parent.embed.title")).
									Build(),
							).
							SetContainerComponents(
								discord.NewActionRow(
									discord.ButtonComponent{
										Label:    translate.Message(event.Locale(), "component.game.cinchiro.game.roll.button.label"),
										Style:    discord.ButtonStylePrimary,
										CustomID: fmt.Sprintf("game:chinchirorin-roll:%s", session.ID),
									},
								),
							).
							BuildCreate(),
						); err != nil {
							return errors.NewError(err)
						}
					}
				}

				return nil
			},
		},
	}).SetComponent(c)
}
