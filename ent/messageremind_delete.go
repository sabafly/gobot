// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/sabafly/gobot/ent/messageremind"
	"github.com/sabafly/gobot/ent/predicate"
)

// MessageRemindDelete is the builder for deleting a MessageRemind entity.
type MessageRemindDelete struct {
	config
	hooks    []Hook
	mutation *MessageRemindMutation
}

// Where appends a list predicates to the MessageRemindDelete builder.
func (mrd *MessageRemindDelete) Where(ps ...predicate.MessageRemind) *MessageRemindDelete {
	mrd.mutation.Where(ps...)
	return mrd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (mrd *MessageRemindDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, mrd.sqlExec, mrd.mutation, mrd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (mrd *MessageRemindDelete) ExecX(ctx context.Context) int {
	n, err := mrd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (mrd *MessageRemindDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(messageremind.Table, sqlgraph.NewFieldSpec(messageremind.FieldID, field.TypeUUID))
	if ps := mrd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, mrd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	mrd.mutation.done = true
	return affected, err
}

// MessageRemindDeleteOne is the builder for deleting a single MessageRemind entity.
type MessageRemindDeleteOne struct {
	mrd *MessageRemindDelete
}

// Where appends a list predicates to the MessageRemindDelete builder.
func (mrdo *MessageRemindDeleteOne) Where(ps ...predicate.MessageRemind) *MessageRemindDeleteOne {
	mrdo.mrd.mutation.Where(ps...)
	return mrdo
}

// Exec executes the deletion query.
func (mrdo *MessageRemindDeleteOne) Exec(ctx context.Context) error {
	n, err := mrdo.mrd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{messageremind.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (mrdo *MessageRemindDeleteOne) ExecX(ctx context.Context) {
	if err := mrdo.Exec(ctx); err != nil {
		panic(err)
	}
}