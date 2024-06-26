// Code generated by ent, DO NOT EDIT.

package chinchiroplayer

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldLTE(FieldID, id))
}

// Point applies equality check predicate on the "point" field. It's identical to PointEQ.
func Point(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldPoint, v))
}

// IsOwner applies equality check predicate on the "is_owner" field. It's identical to IsOwnerEQ.
func IsOwner(v bool) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldIsOwner, v))
}

// UserID applies equality check predicate on the "user_id" field. It's identical to UserIDEQ.
func UserID(v snowflake.ID) predicate.ChinchiroPlayer {
	vc := uint64(v)
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldUserID, vc))
}

// Bet applies equality check predicate on the "bet" field. It's identical to BetEQ.
func Bet(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldBet, v))
}

// PointEQ applies the EQ predicate on the "point" field.
func PointEQ(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldPoint, v))
}

// PointNEQ applies the NEQ predicate on the "point" field.
func PointNEQ(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNEQ(FieldPoint, v))
}

// PointIn applies the In predicate on the "point" field.
func PointIn(vs ...int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldIn(FieldPoint, vs...))
}

// PointNotIn applies the NotIn predicate on the "point" field.
func PointNotIn(vs ...int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNotIn(FieldPoint, vs...))
}

// PointGT applies the GT predicate on the "point" field.
func PointGT(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldGT(FieldPoint, v))
}

// PointGTE applies the GTE predicate on the "point" field.
func PointGTE(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldGTE(FieldPoint, v))
}

// PointLT applies the LT predicate on the "point" field.
func PointLT(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldLT(FieldPoint, v))
}

// PointLTE applies the LTE predicate on the "point" field.
func PointLTE(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldLTE(FieldPoint, v))
}

// IsOwnerEQ applies the EQ predicate on the "is_owner" field.
func IsOwnerEQ(v bool) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldIsOwner, v))
}

// IsOwnerNEQ applies the NEQ predicate on the "is_owner" field.
func IsOwnerNEQ(v bool) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNEQ(FieldIsOwner, v))
}

// UserIDEQ applies the EQ predicate on the "user_id" field.
func UserIDEQ(v snowflake.ID) predicate.ChinchiroPlayer {
	vc := uint64(v)
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldUserID, vc))
}

// UserIDNEQ applies the NEQ predicate on the "user_id" field.
func UserIDNEQ(v snowflake.ID) predicate.ChinchiroPlayer {
	vc := uint64(v)
	return predicate.ChinchiroPlayer(sql.FieldNEQ(FieldUserID, vc))
}

// UserIDIn applies the In predicate on the "user_id" field.
func UserIDIn(vs ...snowflake.ID) predicate.ChinchiroPlayer {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.ChinchiroPlayer(sql.FieldIn(FieldUserID, v...))
}

// UserIDNotIn applies the NotIn predicate on the "user_id" field.
func UserIDNotIn(vs ...snowflake.ID) predicate.ChinchiroPlayer {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = uint64(vs[i])
	}
	return predicate.ChinchiroPlayer(sql.FieldNotIn(FieldUserID, v...))
}

// BetEQ applies the EQ predicate on the "bet" field.
func BetEQ(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldEQ(FieldBet, v))
}

// BetNEQ applies the NEQ predicate on the "bet" field.
func BetNEQ(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNEQ(FieldBet, v))
}

// BetIn applies the In predicate on the "bet" field.
func BetIn(vs ...int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldIn(FieldBet, vs...))
}

// BetNotIn applies the NotIn predicate on the "bet" field.
func BetNotIn(vs ...int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNotIn(FieldBet, vs...))
}

// BetGT applies the GT predicate on the "bet" field.
func BetGT(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldGT(FieldBet, v))
}

// BetGTE applies the GTE predicate on the "bet" field.
func BetGTE(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldGTE(FieldBet, v))
}

// BetLT applies the LT predicate on the "bet" field.
func BetLT(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldLT(FieldBet, v))
}

// BetLTE applies the LTE predicate on the "bet" field.
func BetLTE(v int) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldLTE(FieldBet, v))
}

// BetIsNil applies the IsNil predicate on the "bet" field.
func BetIsNil() predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldIsNull(FieldBet))
}

// BetNotNil applies the NotNil predicate on the "bet" field.
func BetNotNil() predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNotNull(FieldBet))
}

// DicesIsNil applies the IsNil predicate on the "dices" field.
func DicesIsNil() predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldIsNull(FieldDices))
}

// DicesNotNil applies the NotNil predicate on the "dices" field.
func DicesNotNil() predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.FieldNotNull(FieldDices))
}

// HasUser applies the HasEdge predicate on the "user" edge.
func HasUser() predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, UserTable, UserColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasUserWith applies the HasEdge predicate on the "user" edge with a given conditions (other predicates).
func HasUserWith(preds ...predicate.User) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(func(s *sql.Selector) {
		step := newUserStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasSession applies the HasEdge predicate on the "session" edge.
func HasSession() predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, SessionTable, SessionColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasSessionWith applies the HasEdge predicate on the "session" edge with a given conditions (other predicates).
func HasSessionWith(preds ...predicate.ChinchiroSession) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(func(s *sql.Selector) {
		step := newSessionStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.ChinchiroPlayer) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.ChinchiroPlayer) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.ChinchiroPlayer) predicate.ChinchiroPlayer {
	return predicate.ChinchiroPlayer(sql.NotPredicates(p))
}
