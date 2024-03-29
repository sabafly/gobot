// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
)

// RolePanelPlacedDelete is the builder for deleting a RolePanelPlaced entity.
type RolePanelPlacedDelete struct {
	config
	hooks    []Hook
	mutation *RolePanelPlacedMutation
}

// Where appends a list predicates to the RolePanelPlacedDelete builder.
func (rppd *RolePanelPlacedDelete) Where(ps ...predicate.RolePanelPlaced) *RolePanelPlacedDelete {
	rppd.mutation.Where(ps...)
	return rppd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (rppd *RolePanelPlacedDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, rppd.sqlExec, rppd.mutation, rppd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (rppd *RolePanelPlacedDelete) ExecX(ctx context.Context) int {
	n, err := rppd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (rppd *RolePanelPlacedDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(rolepanelplaced.Table, sqlgraph.NewFieldSpec(rolepanelplaced.FieldID, field.TypeUUID))
	if ps := rppd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, rppd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	rppd.mutation.done = true
	return affected, err
}

// RolePanelPlacedDeleteOne is the builder for deleting a single RolePanelPlaced entity.
type RolePanelPlacedDeleteOne struct {
	rppd *RolePanelPlacedDelete
}

// Where appends a list predicates to the RolePanelPlacedDelete builder.
func (rppdo *RolePanelPlacedDeleteOne) Where(ps ...predicate.RolePanelPlaced) *RolePanelPlacedDeleteOne {
	rppdo.rppd.mutation.Where(ps...)
	return rppdo
}

// Exec executes the deletion query.
func (rppdo *RolePanelPlacedDeleteOne) Exec(ctx context.Context) error {
	n, err := rppdo.rppd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{rolepanelplaced.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (rppdo *RolePanelPlacedDeleteOne) ExecX(ctx context.Context) {
	if err := rppdo.Exec(ctx); err != nil {
		panic(err)
	}
}
