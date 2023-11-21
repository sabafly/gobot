// Code generated by ent, DO NOT EDIT.

package rolepanelplaced

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldLTE(FieldID, id))
}

// MessageID applies equality check predicate on the "message_id" field. It's identical to MessageIDEQ.
func MessageID(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldMessageID, vc))
}

// ChannelID applies equality check predicate on the "channel_id" field. It's identical to ChannelIDEQ.
func ChannelID(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldChannelID, vc))
}

// ButtonType applies equality check predicate on the "button_type" field. It's identical to ButtonTypeEQ.
func ButtonType(v discord.ButtonStyle) predicate.RolePanelPlaced {
	vc := int(v)
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldButtonType, vc))
}

// ShowName applies equality check predicate on the "show_name" field. It's identical to ShowNameEQ.
func ShowName(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldShowName, v))
}

// FoldingSelectMenu applies equality check predicate on the "folding_select_menu" field. It's identical to FoldingSelectMenuEQ.
func FoldingSelectMenu(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldFoldingSelectMenu, v))
}

// HideNotice applies equality check predicate on the "hide_notice" field. It's identical to HideNoticeEQ.
func HideNotice(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldHideNotice, v))
}

// UseDisplayName applies equality check predicate on the "use_display_name" field. It's identical to UseDisplayNameEQ.
func UseDisplayName(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldUseDisplayName, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldCreatedAt, v))
}

// Uses applies equality check predicate on the "uses" field. It's identical to UsesEQ.
func Uses(v int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldUses, v))
}

// MessageIDEQ applies the EQ predicate on the "message_id" field.
func MessageIDEQ(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldMessageID, vc))
}

// MessageIDNEQ applies the NEQ predicate on the "message_id" field.
func MessageIDNEQ(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldMessageID, vc))
}

// MessageIDIn applies the In predicate on the "message_id" field.
func MessageIDIn(vs ...snowflake.ID) predicate.RolePanelPlaced {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.RolePanelPlaced(sql.FieldIn(FieldMessageID, v...))
}

// MessageIDNotIn applies the NotIn predicate on the "message_id" field.
func MessageIDNotIn(vs ...snowflake.ID) predicate.RolePanelPlaced {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.RolePanelPlaced(sql.FieldNotIn(FieldMessageID, v...))
}

// MessageIDGT applies the GT predicate on the "message_id" field.
func MessageIDGT(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldGT(FieldMessageID, vc))
}

// MessageIDGTE applies the GTE predicate on the "message_id" field.
func MessageIDGTE(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldGTE(FieldMessageID, vc))
}

// MessageIDLT applies the LT predicate on the "message_id" field.
func MessageIDLT(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldLT(FieldMessageID, vc))
}

// MessageIDLTE applies the LTE predicate on the "message_id" field.
func MessageIDLTE(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldLTE(FieldMessageID, vc))
}

// MessageIDIsNil applies the IsNil predicate on the "message_id" field.
func MessageIDIsNil() predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldIsNull(FieldMessageID))
}

// MessageIDNotNil applies the NotNil predicate on the "message_id" field.
func MessageIDNotNil() predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNotNull(FieldMessageID))
}

// ChannelIDEQ applies the EQ predicate on the "channel_id" field.
func ChannelIDEQ(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldChannelID, vc))
}

// ChannelIDNEQ applies the NEQ predicate on the "channel_id" field.
func ChannelIDNEQ(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldChannelID, vc))
}

// ChannelIDIn applies the In predicate on the "channel_id" field.
func ChannelIDIn(vs ...snowflake.ID) predicate.RolePanelPlaced {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.RolePanelPlaced(sql.FieldIn(FieldChannelID, v...))
}

// ChannelIDNotIn applies the NotIn predicate on the "channel_id" field.
func ChannelIDNotIn(vs ...snowflake.ID) predicate.RolePanelPlaced {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.RolePanelPlaced(sql.FieldNotIn(FieldChannelID, v...))
}

// ChannelIDGT applies the GT predicate on the "channel_id" field.
func ChannelIDGT(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldGT(FieldChannelID, vc))
}

// ChannelIDGTE applies the GTE predicate on the "channel_id" field.
func ChannelIDGTE(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldGTE(FieldChannelID, vc))
}

// ChannelIDLT applies the LT predicate on the "channel_id" field.
func ChannelIDLT(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldLT(FieldChannelID, vc))
}

// ChannelIDLTE applies the LTE predicate on the "channel_id" field.
func ChannelIDLTE(v snowflake.ID) predicate.RolePanelPlaced {
	vc := uint64(v)
	return predicate.RolePanelPlaced(sql.FieldLTE(FieldChannelID, vc))
}

// TypeEQ applies the EQ predicate on the "type" field.
func TypeEQ(v Type) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldType, v))
}

// TypeNEQ applies the NEQ predicate on the "type" field.
func TypeNEQ(v Type) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldType, v))
}

// TypeIn applies the In predicate on the "type" field.
func TypeIn(vs ...Type) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldIn(FieldType, vs...))
}

// TypeNotIn applies the NotIn predicate on the "type" field.
func TypeNotIn(vs ...Type) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNotIn(FieldType, vs...))
}

// TypeIsNil applies the IsNil predicate on the "type" field.
func TypeIsNil() predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldIsNull(FieldType))
}

// TypeNotNil applies the NotNil predicate on the "type" field.
func TypeNotNil() predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNotNull(FieldType))
}

// ButtonTypeEQ applies the EQ predicate on the "button_type" field.
func ButtonTypeEQ(v discord.ButtonStyle) predicate.RolePanelPlaced {
	vc := int(v)
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldButtonType, vc))
}

// ButtonTypeNEQ applies the NEQ predicate on the "button_type" field.
func ButtonTypeNEQ(v discord.ButtonStyle) predicate.RolePanelPlaced {
	vc := int(v)
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldButtonType, vc))
}

// ButtonTypeIn applies the In predicate on the "button_type" field.
func ButtonTypeIn(vs ...discord.ButtonStyle) predicate.RolePanelPlaced {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = int(vs[i])
	}
	return predicate.RolePanelPlaced(sql.FieldIn(FieldButtonType, v...))
}

// ButtonTypeNotIn applies the NotIn predicate on the "button_type" field.
func ButtonTypeNotIn(vs ...discord.ButtonStyle) predicate.RolePanelPlaced {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = int(vs[i])
	}
	return predicate.RolePanelPlaced(sql.FieldNotIn(FieldButtonType, v...))
}

// ButtonTypeGT applies the GT predicate on the "button_type" field.
func ButtonTypeGT(v discord.ButtonStyle) predicate.RolePanelPlaced {
	vc := int(v)
	return predicate.RolePanelPlaced(sql.FieldGT(FieldButtonType, vc))
}

// ButtonTypeGTE applies the GTE predicate on the "button_type" field.
func ButtonTypeGTE(v discord.ButtonStyle) predicate.RolePanelPlaced {
	vc := int(v)
	return predicate.RolePanelPlaced(sql.FieldGTE(FieldButtonType, vc))
}

// ButtonTypeLT applies the LT predicate on the "button_type" field.
func ButtonTypeLT(v discord.ButtonStyle) predicate.RolePanelPlaced {
	vc := int(v)
	return predicate.RolePanelPlaced(sql.FieldLT(FieldButtonType, vc))
}

// ButtonTypeLTE applies the LTE predicate on the "button_type" field.
func ButtonTypeLTE(v discord.ButtonStyle) predicate.RolePanelPlaced {
	vc := int(v)
	return predicate.RolePanelPlaced(sql.FieldLTE(FieldButtonType, vc))
}

// ShowNameEQ applies the EQ predicate on the "show_name" field.
func ShowNameEQ(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldShowName, v))
}

// ShowNameNEQ applies the NEQ predicate on the "show_name" field.
func ShowNameNEQ(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldShowName, v))
}

// FoldingSelectMenuEQ applies the EQ predicate on the "folding_select_menu" field.
func FoldingSelectMenuEQ(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldFoldingSelectMenu, v))
}

// FoldingSelectMenuNEQ applies the NEQ predicate on the "folding_select_menu" field.
func FoldingSelectMenuNEQ(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldFoldingSelectMenu, v))
}

// HideNoticeEQ applies the EQ predicate on the "hide_notice" field.
func HideNoticeEQ(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldHideNotice, v))
}

// HideNoticeNEQ applies the NEQ predicate on the "hide_notice" field.
func HideNoticeNEQ(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldHideNotice, v))
}

// UseDisplayNameEQ applies the EQ predicate on the "use_display_name" field.
func UseDisplayNameEQ(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldUseDisplayName, v))
}

// UseDisplayNameNEQ applies the NEQ predicate on the "use_display_name" field.
func UseDisplayNameNEQ(v bool) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldUseDisplayName, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldLTE(FieldCreatedAt, v))
}

// UsesEQ applies the EQ predicate on the "uses" field.
func UsesEQ(v int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldEQ(FieldUses, v))
}

// UsesNEQ applies the NEQ predicate on the "uses" field.
func UsesNEQ(v int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNEQ(FieldUses, v))
}

// UsesIn applies the In predicate on the "uses" field.
func UsesIn(vs ...int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldIn(FieldUses, vs...))
}

// UsesNotIn applies the NotIn predicate on the "uses" field.
func UsesNotIn(vs ...int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldNotIn(FieldUses, vs...))
}

// UsesGT applies the GT predicate on the "uses" field.
func UsesGT(v int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldGT(FieldUses, v))
}

// UsesGTE applies the GTE predicate on the "uses" field.
func UsesGTE(v int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldGTE(FieldUses, v))
}

// UsesLT applies the LT predicate on the "uses" field.
func UsesLT(v int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldLT(FieldUses, v))
}

// UsesLTE applies the LTE predicate on the "uses" field.
func UsesLTE(v int) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.FieldLTE(FieldUses, v))
}

// HasGuild applies the HasEdge predicate on the "guild" edge.
func HasGuild() predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, GuildTable, GuildColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasGuildWith applies the HasEdge predicate on the "guild" edge with a given conditions (other predicates).
func HasGuildWith(preds ...predicate.Guild) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(func(s *sql.Selector) {
		step := newGuildStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasRolePanel applies the HasEdge predicate on the "role_panel" edge.
func HasRolePanel() predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, RolePanelTable, RolePanelColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasRolePanelWith applies the HasEdge predicate on the "role_panel" edge with a given conditions (other predicates).
func HasRolePanelWith(preds ...predicate.RolePanel) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(func(s *sql.Selector) {
		step := newRolePanelStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.RolePanelPlaced) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.RolePanelPlaced) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.RolePanelPlaced) predicate.RolePanelPlaced {
	return predicate.RolePanelPlaced(sql.NotPredicates(p))
}
