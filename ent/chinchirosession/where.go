// Code generated by ent, DO NOT EDIT.

package chinchirosession

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldLTE(FieldID, id))
}

// Turn applies equality check predicate on the "turn" field. It's identical to TurnEQ.
func Turn(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldEQ(FieldTurn, v))
}

// Loop applies equality check predicate on the "loop" field. It's identical to LoopEQ.
func Loop(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldEQ(FieldLoop, v))
}

// TurnEQ applies the EQ predicate on the "turn" field.
func TurnEQ(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldEQ(FieldTurn, v))
}

// TurnNEQ applies the NEQ predicate on the "turn" field.
func TurnNEQ(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldNEQ(FieldTurn, v))
}

// TurnIn applies the In predicate on the "turn" field.
func TurnIn(vs ...int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldIn(FieldTurn, vs...))
}

// TurnNotIn applies the NotIn predicate on the "turn" field.
func TurnNotIn(vs ...int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldNotIn(FieldTurn, vs...))
}

// TurnGT applies the GT predicate on the "turn" field.
func TurnGT(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldGT(FieldTurn, v))
}

// TurnGTE applies the GTE predicate on the "turn" field.
func TurnGTE(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldGTE(FieldTurn, v))
}

// TurnLT applies the LT predicate on the "turn" field.
func TurnLT(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldLT(FieldTurn, v))
}

// TurnLTE applies the LTE predicate on the "turn" field.
func TurnLTE(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldLTE(FieldTurn, v))
}

// LoopEQ applies the EQ predicate on the "loop" field.
func LoopEQ(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldEQ(FieldLoop, v))
}

// LoopNEQ applies the NEQ predicate on the "loop" field.
func LoopNEQ(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldNEQ(FieldLoop, v))
}

// LoopIn applies the In predicate on the "loop" field.
func LoopIn(vs ...int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldIn(FieldLoop, vs...))
}

// LoopNotIn applies the NotIn predicate on the "loop" field.
func LoopNotIn(vs ...int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldNotIn(FieldLoop, vs...))
}

// LoopGT applies the GT predicate on the "loop" field.
func LoopGT(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldGT(FieldLoop, v))
}

// LoopGTE applies the GTE predicate on the "loop" field.
func LoopGTE(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldGTE(FieldLoop, v))
}

// LoopLT applies the LT predicate on the "loop" field.
func LoopLT(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldLT(FieldLoop, v))
}

// LoopLTE applies the LTE predicate on the "loop" field.
func LoopLTE(v int) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.FieldLTE(FieldLoop, v))
}

// HasGuild applies the HasEdge predicate on the "guild" edge.
func HasGuild() predicate.ChinchiroSession {
	return predicate.ChinchiroSession(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, GuildTable, GuildColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasGuildWith applies the HasEdge predicate on the "guild" edge with a given conditions (other predicates).
func HasGuildWith(preds ...predicate.Guild) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(func(s *sql.Selector) {
		step := newGuildStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasPlayers applies the HasEdge predicate on the "players" edge.
func HasPlayers() predicate.ChinchiroSession {
	return predicate.ChinchiroSession(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, PlayersTable, PlayersColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlayersWith applies the HasEdge predicate on the "players" edge with a given conditions (other predicates).
func HasPlayersWith(preds ...predicate.ChinchiroPlayer) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(func(s *sql.Selector) {
		step := newPlayersStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.ChinchiroSession) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.ChinchiroSession) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.ChinchiroSession) predicate.ChinchiroSession {
	return predicate.ChinchiroSession(sql.NotPredicates(p))
}
