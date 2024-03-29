// Code generated by ent, DO NOT EDIT.

package messageremind

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldLTE(FieldID, id))
}

// ChannelID applies equality check predicate on the "channel_id" field. It's identical to ChannelIDEQ.
func ChannelID(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldEQ(FieldChannelID, vc))
}

// AuthorID applies equality check predicate on the "author_id" field. It's identical to AuthorIDEQ.
func AuthorID(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldEQ(FieldAuthorID, vc))
}

// Time applies equality check predicate on the "time" field. It's identical to TimeEQ.
func Time(v time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEQ(FieldTime, v))
}

// Content applies equality check predicate on the "content" field. It's identical to ContentEQ.
func Content(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEQ(FieldContent, v))
}

// Name applies equality check predicate on the "name" field. It's identical to NameEQ.
func Name(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEQ(FieldName, v))
}

// ChannelIDEQ applies the EQ predicate on the "channel_id" field.
func ChannelIDEQ(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldEQ(FieldChannelID, vc))
}

// ChannelIDNEQ applies the NEQ predicate on the "channel_id" field.
func ChannelIDNEQ(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldNEQ(FieldChannelID, vc))
}

// ChannelIDIn applies the In predicate on the "channel_id" field.
func ChannelIDIn(vs ...snowflake.ID) predicate.MessageRemind {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.MessageRemind(sql.FieldIn(FieldChannelID, v...))
}

// ChannelIDNotIn applies the NotIn predicate on the "channel_id" field.
func ChannelIDNotIn(vs ...snowflake.ID) predicate.MessageRemind {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.MessageRemind(sql.FieldNotIn(FieldChannelID, v...))
}

// ChannelIDGT applies the GT predicate on the "channel_id" field.
func ChannelIDGT(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldGT(FieldChannelID, vc))
}

// ChannelIDGTE applies the GTE predicate on the "channel_id" field.
func ChannelIDGTE(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldGTE(FieldChannelID, vc))
}

// ChannelIDLT applies the LT predicate on the "channel_id" field.
func ChannelIDLT(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldLT(FieldChannelID, vc))
}

// ChannelIDLTE applies the LTE predicate on the "channel_id" field.
func ChannelIDLTE(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldLTE(FieldChannelID, vc))
}

// AuthorIDEQ applies the EQ predicate on the "author_id" field.
func AuthorIDEQ(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldEQ(FieldAuthorID, vc))
}

// AuthorIDNEQ applies the NEQ predicate on the "author_id" field.
func AuthorIDNEQ(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldNEQ(FieldAuthorID, vc))
}

// AuthorIDIn applies the In predicate on the "author_id" field.
func AuthorIDIn(vs ...snowflake.ID) predicate.MessageRemind {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.MessageRemind(sql.FieldIn(FieldAuthorID, v...))
}

// AuthorIDNotIn applies the NotIn predicate on the "author_id" field.
func AuthorIDNotIn(vs ...snowflake.ID) predicate.MessageRemind {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.MessageRemind(sql.FieldNotIn(FieldAuthorID, v...))
}

// AuthorIDGT applies the GT predicate on the "author_id" field.
func AuthorIDGT(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldGT(FieldAuthorID, vc))
}

// AuthorIDGTE applies the GTE predicate on the "author_id" field.
func AuthorIDGTE(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldGTE(FieldAuthorID, vc))
}

// AuthorIDLT applies the LT predicate on the "author_id" field.
func AuthorIDLT(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldLT(FieldAuthorID, vc))
}

// AuthorIDLTE applies the LTE predicate on the "author_id" field.
func AuthorIDLTE(v snowflake.ID) predicate.MessageRemind {
	vc := uint64(v)
	return predicate.MessageRemind(sql.FieldLTE(FieldAuthorID, vc))
}

// TimeEQ applies the EQ predicate on the "time" field.
func TimeEQ(v time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEQ(FieldTime, v))
}

// TimeNEQ applies the NEQ predicate on the "time" field.
func TimeNEQ(v time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldNEQ(FieldTime, v))
}

// TimeIn applies the In predicate on the "time" field.
func TimeIn(vs ...time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldIn(FieldTime, vs...))
}

// TimeNotIn applies the NotIn predicate on the "time" field.
func TimeNotIn(vs ...time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldNotIn(FieldTime, vs...))
}

// TimeGT applies the GT predicate on the "time" field.
func TimeGT(v time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldGT(FieldTime, v))
}

// TimeGTE applies the GTE predicate on the "time" field.
func TimeGTE(v time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldGTE(FieldTime, v))
}

// TimeLT applies the LT predicate on the "time" field.
func TimeLT(v time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldLT(FieldTime, v))
}

// TimeLTE applies the LTE predicate on the "time" field.
func TimeLTE(v time.Time) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldLTE(FieldTime, v))
}

// ContentEQ applies the EQ predicate on the "content" field.
func ContentEQ(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEQ(FieldContent, v))
}

// ContentNEQ applies the NEQ predicate on the "content" field.
func ContentNEQ(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldNEQ(FieldContent, v))
}

// ContentIn applies the In predicate on the "content" field.
func ContentIn(vs ...string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldIn(FieldContent, vs...))
}

// ContentNotIn applies the NotIn predicate on the "content" field.
func ContentNotIn(vs ...string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldNotIn(FieldContent, vs...))
}

// ContentGT applies the GT predicate on the "content" field.
func ContentGT(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldGT(FieldContent, v))
}

// ContentGTE applies the GTE predicate on the "content" field.
func ContentGTE(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldGTE(FieldContent, v))
}

// ContentLT applies the LT predicate on the "content" field.
func ContentLT(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldLT(FieldContent, v))
}

// ContentLTE applies the LTE predicate on the "content" field.
func ContentLTE(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldLTE(FieldContent, v))
}

// ContentContains applies the Contains predicate on the "content" field.
func ContentContains(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldContains(FieldContent, v))
}

// ContentHasPrefix applies the HasPrefix predicate on the "content" field.
func ContentHasPrefix(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldHasPrefix(FieldContent, v))
}

// ContentHasSuffix applies the HasSuffix predicate on the "content" field.
func ContentHasSuffix(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldHasSuffix(FieldContent, v))
}

// ContentEqualFold applies the EqualFold predicate on the "content" field.
func ContentEqualFold(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEqualFold(FieldContent, v))
}

// ContentContainsFold applies the ContainsFold predicate on the "content" field.
func ContentContainsFold(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldContainsFold(FieldContent, v))
}

// NameEQ applies the EQ predicate on the "name" field.
func NameEQ(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEQ(FieldName, v))
}

// NameNEQ applies the NEQ predicate on the "name" field.
func NameNEQ(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldNEQ(FieldName, v))
}

// NameIn applies the In predicate on the "name" field.
func NameIn(vs ...string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldIn(FieldName, vs...))
}

// NameNotIn applies the NotIn predicate on the "name" field.
func NameNotIn(vs ...string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldNotIn(FieldName, vs...))
}

// NameGT applies the GT predicate on the "name" field.
func NameGT(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldGT(FieldName, v))
}

// NameGTE applies the GTE predicate on the "name" field.
func NameGTE(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldGTE(FieldName, v))
}

// NameLT applies the LT predicate on the "name" field.
func NameLT(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldLT(FieldName, v))
}

// NameLTE applies the LTE predicate on the "name" field.
func NameLTE(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldLTE(FieldName, v))
}

// NameContains applies the Contains predicate on the "name" field.
func NameContains(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldContains(FieldName, v))
}

// NameHasPrefix applies the HasPrefix predicate on the "name" field.
func NameHasPrefix(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldHasPrefix(FieldName, v))
}

// NameHasSuffix applies the HasSuffix predicate on the "name" field.
func NameHasSuffix(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldHasSuffix(FieldName, v))
}

// NameEqualFold applies the EqualFold predicate on the "name" field.
func NameEqualFold(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldEqualFold(FieldName, v))
}

// NameContainsFold applies the ContainsFold predicate on the "name" field.
func NameContainsFold(v string) predicate.MessageRemind {
	return predicate.MessageRemind(sql.FieldContainsFold(FieldName, v))
}

// HasGuild applies the HasEdge predicate on the "guild" edge.
func HasGuild() predicate.MessageRemind {
	return predicate.MessageRemind(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, GuildTable, GuildColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasGuildWith applies the HasEdge predicate on the "guild" edge with a given conditions (other predicates).
func HasGuildWith(preds ...predicate.Guild) predicate.MessageRemind {
	return predicate.MessageRemind(func(s *sql.Selector) {
		step := newGuildStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.MessageRemind) predicate.MessageRemind {
	return predicate.MessageRemind(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.MessageRemind) predicate.MessageRemind {
	return predicate.MessageRemind(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.MessageRemind) predicate.MessageRemind {
	return predicate.MessageRemind(sql.NotPredicates(p))
}
