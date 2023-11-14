// Code generated by ent, DO NOT EDIT.

package guild

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id snowflake.ID) predicate.Guild {
	return predicate.Guild(sql.FieldLTE(FieldID, id))
}

// Name applies equality check predicate on the "name" field. It's identical to NameEQ.
func Name(v string) predicate.Guild {
	return predicate.Guild(sql.FieldEQ(FieldName, v))
}

// Locale applies equality check predicate on the "locale" field. It's identical to LocaleEQ.
func Locale(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldEQ(FieldLocale, vc))
}

// NameEQ applies the EQ predicate on the "name" field.
func NameEQ(v string) predicate.Guild {
	return predicate.Guild(sql.FieldEQ(FieldName, v))
}

// NameNEQ applies the NEQ predicate on the "name" field.
func NameNEQ(v string) predicate.Guild {
	return predicate.Guild(sql.FieldNEQ(FieldName, v))
}

// NameIn applies the In predicate on the "name" field.
func NameIn(vs ...string) predicate.Guild {
	return predicate.Guild(sql.FieldIn(FieldName, vs...))
}

// NameNotIn applies the NotIn predicate on the "name" field.
func NameNotIn(vs ...string) predicate.Guild {
	return predicate.Guild(sql.FieldNotIn(FieldName, vs...))
}

// NameGT applies the GT predicate on the "name" field.
func NameGT(v string) predicate.Guild {
	return predicate.Guild(sql.FieldGT(FieldName, v))
}

// NameGTE applies the GTE predicate on the "name" field.
func NameGTE(v string) predicate.Guild {
	return predicate.Guild(sql.FieldGTE(FieldName, v))
}

// NameLT applies the LT predicate on the "name" field.
func NameLT(v string) predicate.Guild {
	return predicate.Guild(sql.FieldLT(FieldName, v))
}

// NameLTE applies the LTE predicate on the "name" field.
func NameLTE(v string) predicate.Guild {
	return predicate.Guild(sql.FieldLTE(FieldName, v))
}

// NameContains applies the Contains predicate on the "name" field.
func NameContains(v string) predicate.Guild {
	return predicate.Guild(sql.FieldContains(FieldName, v))
}

// NameHasPrefix applies the HasPrefix predicate on the "name" field.
func NameHasPrefix(v string) predicate.Guild {
	return predicate.Guild(sql.FieldHasPrefix(FieldName, v))
}

// NameHasSuffix applies the HasSuffix predicate on the "name" field.
func NameHasSuffix(v string) predicate.Guild {
	return predicate.Guild(sql.FieldHasSuffix(FieldName, v))
}

// NameEqualFold applies the EqualFold predicate on the "name" field.
func NameEqualFold(v string) predicate.Guild {
	return predicate.Guild(sql.FieldEqualFold(FieldName, v))
}

// NameContainsFold applies the ContainsFold predicate on the "name" field.
func NameContainsFold(v string) predicate.Guild {
	return predicate.Guild(sql.FieldContainsFold(FieldName, v))
}

// LocaleEQ applies the EQ predicate on the "locale" field.
func LocaleEQ(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldEQ(FieldLocale, vc))
}

// LocaleNEQ applies the NEQ predicate on the "locale" field.
func LocaleNEQ(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldNEQ(FieldLocale, vc))
}

// LocaleIn applies the In predicate on the "locale" field.
func LocaleIn(vs ...discord.Locale) predicate.Guild {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = string(vs[i])
	}
	return predicate.Guild(sql.FieldIn(FieldLocale, v...))
}

// LocaleNotIn applies the NotIn predicate on the "locale" field.
func LocaleNotIn(vs ...discord.Locale) predicate.Guild {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = string(vs[i])
	}
	return predicate.Guild(sql.FieldNotIn(FieldLocale, v...))
}

// LocaleGT applies the GT predicate on the "locale" field.
func LocaleGT(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldGT(FieldLocale, vc))
}

// LocaleGTE applies the GTE predicate on the "locale" field.
func LocaleGTE(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldGTE(FieldLocale, vc))
}

// LocaleLT applies the LT predicate on the "locale" field.
func LocaleLT(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldLT(FieldLocale, vc))
}

// LocaleLTE applies the LTE predicate on the "locale" field.
func LocaleLTE(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldLTE(FieldLocale, vc))
}

// LocaleContains applies the Contains predicate on the "locale" field.
func LocaleContains(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldContains(FieldLocale, vc))
}

// LocaleHasPrefix applies the HasPrefix predicate on the "locale" field.
func LocaleHasPrefix(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldHasPrefix(FieldLocale, vc))
}

// LocaleHasSuffix applies the HasSuffix predicate on the "locale" field.
func LocaleHasSuffix(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldHasSuffix(FieldLocale, vc))
}

// LocaleEqualFold applies the EqualFold predicate on the "locale" field.
func LocaleEqualFold(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldEqualFold(FieldLocale, vc))
}

// LocaleContainsFold applies the ContainsFold predicate on the "locale" field.
func LocaleContainsFold(v discord.Locale) predicate.Guild {
	vc := string(v)
	return predicate.Guild(sql.FieldContainsFold(FieldLocale, vc))
}

// HasOwner applies the HasEdge predicate on the "owner" edge.
func HasOwner() predicate.Guild {
	return predicate.Guild(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, OwnerTable, OwnerColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasOwnerWith applies the HasEdge predicate on the "owner" edge with a given conditions (other predicates).
func HasOwnerWith(preds ...predicate.User) predicate.Guild {
	return predicate.Guild(func(s *sql.Selector) {
		step := newOwnerStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasMembers applies the HasEdge predicate on the "members" edge.
func HasMembers() predicate.Guild {
	return predicate.Guild(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, MembersTable, MembersPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasMembersWith applies the HasEdge predicate on the "members" edge with a given conditions (other predicates).
func HasMembersWith(preds ...predicate.Member) predicate.Guild {
	return predicate.Guild(func(s *sql.Selector) {
		step := newMembersStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Guild) predicate.Guild {
	return predicate.Guild(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Guild) predicate.Guild {
	return predicate.Guild(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Guild) predicate.Guild {
	return predicate.Guild(sql.NotPredicates(p))
}