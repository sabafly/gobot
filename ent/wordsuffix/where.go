// Code generated by ent, DO NOT EDIT.

package wordsuffix

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldLTE(FieldID, id))
}

// Suffix applies equality check predicate on the "suffix" field. It's identical to SuffixEQ.
func Suffix(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldEQ(FieldSuffix, v))
}

// Expired applies equality check predicate on the "expired" field. It's identical to ExpiredEQ.
func Expired(v time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldEQ(FieldExpired, v))
}

// GuildID applies equality check predicate on the "guild_id" field. It's identical to GuildIDEQ.
func GuildID(v snowflake.ID) predicate.WordSuffix {
	vc := uint64(v)
	return predicate.WordSuffix(sql.FieldEQ(FieldGuildID, vc))
}

// SuffixEQ applies the EQ predicate on the "suffix" field.
func SuffixEQ(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldEQ(FieldSuffix, v))
}

// SuffixNEQ applies the NEQ predicate on the "suffix" field.
func SuffixNEQ(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNEQ(FieldSuffix, v))
}

// SuffixIn applies the In predicate on the "suffix" field.
func SuffixIn(vs ...string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldIn(FieldSuffix, vs...))
}

// SuffixNotIn applies the NotIn predicate on the "suffix" field.
func SuffixNotIn(vs ...string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNotIn(FieldSuffix, vs...))
}

// SuffixGT applies the GT predicate on the "suffix" field.
func SuffixGT(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldGT(FieldSuffix, v))
}

// SuffixGTE applies the GTE predicate on the "suffix" field.
func SuffixGTE(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldGTE(FieldSuffix, v))
}

// SuffixLT applies the LT predicate on the "suffix" field.
func SuffixLT(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldLT(FieldSuffix, v))
}

// SuffixLTE applies the LTE predicate on the "suffix" field.
func SuffixLTE(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldLTE(FieldSuffix, v))
}

// SuffixContains applies the Contains predicate on the "suffix" field.
func SuffixContains(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldContains(FieldSuffix, v))
}

// SuffixHasPrefix applies the HasPrefix predicate on the "suffix" field.
func SuffixHasPrefix(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldHasPrefix(FieldSuffix, v))
}

// SuffixHasSuffix applies the HasSuffix predicate on the "suffix" field.
func SuffixHasSuffix(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldHasSuffix(FieldSuffix, v))
}

// SuffixEqualFold applies the EqualFold predicate on the "suffix" field.
func SuffixEqualFold(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldEqualFold(FieldSuffix, v))
}

// SuffixContainsFold applies the ContainsFold predicate on the "suffix" field.
func SuffixContainsFold(v string) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldContainsFold(FieldSuffix, v))
}

// ExpiredEQ applies the EQ predicate on the "expired" field.
func ExpiredEQ(v time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldEQ(FieldExpired, v))
}

// ExpiredNEQ applies the NEQ predicate on the "expired" field.
func ExpiredNEQ(v time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNEQ(FieldExpired, v))
}

// ExpiredIn applies the In predicate on the "expired" field.
func ExpiredIn(vs ...time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldIn(FieldExpired, vs...))
}

// ExpiredNotIn applies the NotIn predicate on the "expired" field.
func ExpiredNotIn(vs ...time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNotIn(FieldExpired, vs...))
}

// ExpiredGT applies the GT predicate on the "expired" field.
func ExpiredGT(v time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldGT(FieldExpired, v))
}

// ExpiredGTE applies the GTE predicate on the "expired" field.
func ExpiredGTE(v time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldGTE(FieldExpired, v))
}

// ExpiredLT applies the LT predicate on the "expired" field.
func ExpiredLT(v time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldLT(FieldExpired, v))
}

// ExpiredLTE applies the LTE predicate on the "expired" field.
func ExpiredLTE(v time.Time) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldLTE(FieldExpired, v))
}

// ExpiredIsNil applies the IsNil predicate on the "expired" field.
func ExpiredIsNil() predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldIsNull(FieldExpired))
}

// ExpiredNotNil applies the NotNil predicate on the "expired" field.
func ExpiredNotNil() predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNotNull(FieldExpired))
}

// GuildIDEQ applies the EQ predicate on the "guild_id" field.
func GuildIDEQ(v snowflake.ID) predicate.WordSuffix {
	vc := uint64(v)
	return predicate.WordSuffix(sql.FieldEQ(FieldGuildID, vc))
}

// GuildIDNEQ applies the NEQ predicate on the "guild_id" field.
func GuildIDNEQ(v snowflake.ID) predicate.WordSuffix {
	vc := uint64(v)
	return predicate.WordSuffix(sql.FieldNEQ(FieldGuildID, vc))
}

// GuildIDIn applies the In predicate on the "guild_id" field.
func GuildIDIn(vs ...snowflake.ID) predicate.WordSuffix {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.WordSuffix(sql.FieldIn(FieldGuildID, v...))
}

// GuildIDNotIn applies the NotIn predicate on the "guild_id" field.
func GuildIDNotIn(vs ...snowflake.ID) predicate.WordSuffix {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.WordSuffix(sql.FieldNotIn(FieldGuildID, v...))
}

// GuildIDIsNil applies the IsNil predicate on the "guild_id" field.
func GuildIDIsNil() predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldIsNull(FieldGuildID))
}

// GuildIDNotNil applies the NotNil predicate on the "guild_id" field.
func GuildIDNotNil() predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNotNull(FieldGuildID))
}

// RuleEQ applies the EQ predicate on the "rule" field.
func RuleEQ(v Rule) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldEQ(FieldRule, v))
}

// RuleNEQ applies the NEQ predicate on the "rule" field.
func RuleNEQ(v Rule) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNEQ(FieldRule, v))
}

// RuleIn applies the In predicate on the "rule" field.
func RuleIn(vs ...Rule) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldIn(FieldRule, vs...))
}

// RuleNotIn applies the NotIn predicate on the "rule" field.
func RuleNotIn(vs ...Rule) predicate.WordSuffix {
	return predicate.WordSuffix(sql.FieldNotIn(FieldRule, vs...))
}

// HasGuild applies the HasEdge predicate on the "guild" edge.
func HasGuild() predicate.WordSuffix {
	return predicate.WordSuffix(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, GuildTable, GuildColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasGuildWith applies the HasEdge predicate on the "guild" edge with a given conditions (other predicates).
func HasGuildWith(preds ...predicate.Guild) predicate.WordSuffix {
	return predicate.WordSuffix(func(s *sql.Selector) {
		step := newGuildStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasOwner applies the HasEdge predicate on the "owner" edge.
func HasOwner() predicate.WordSuffix {
	return predicate.WordSuffix(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, OwnerTable, OwnerColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasOwnerWith applies the HasEdge predicate on the "owner" edge with a given conditions (other predicates).
func HasOwnerWith(preds ...predicate.User) predicate.WordSuffix {
	return predicate.WordSuffix(func(s *sql.Selector) {
		step := newOwnerStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.WordSuffix) predicate.WordSuffix {
	return predicate.WordSuffix(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.WordSuffix) predicate.WordSuffix {
	return predicate.WordSuffix(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.WordSuffix) predicate.WordSuffix {
	return predicate.WordSuffix(sql.NotPredicates(p))
}
