// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/rolepanel"
)

// RolePanelDelete is the builder for deleting a RolePanel entity.
type RolePanelDelete struct {
	config
	hooks    []Hook
	mutation *RolePanelMutation
}

// Where appends a list predicates to the RolePanelDelete builder.
func (rpd *RolePanelDelete) Where(ps ...predicate.RolePanel) *RolePanelDelete {
	rpd.mutation.Where(ps...)
	return rpd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (rpd *RolePanelDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, rpd.sqlExec, rpd.mutation, rpd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (rpd *RolePanelDelete) ExecX(ctx context.Context) int {
	n, err := rpd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (rpd *RolePanelDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(rolepanel.Table, sqlgraph.NewFieldSpec(rolepanel.FieldID, field.TypeUUID))
	if ps := rpd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, rpd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	rpd.mutation.done = true
	return affected, err
}

// RolePanelDeleteOne is the builder for deleting a single RolePanel entity.
type RolePanelDeleteOne struct {
	rpd *RolePanelDelete
}

// Where appends a list predicates to the RolePanelDelete builder.
func (rpdo *RolePanelDeleteOne) Where(ps ...predicate.RolePanel) *RolePanelDeleteOne {
	rpdo.rpd.mutation.Where(ps...)
	return rpdo
}

// Exec executes the deletion query.
func (rpdo *RolePanelDeleteOne) Exec(ctx context.Context) error {
	n, err := rpdo.rpd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{rolepanel.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (rpdo *RolePanelDeleteOne) ExecX(ctx context.Context) {
	if err := rpdo.Exec(ctx); err != nil {
		panic(err)
	}
}
