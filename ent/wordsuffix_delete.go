// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/wordsuffix"
)

// WordSuffixDelete is the builder for deleting a WordSuffix entity.
type WordSuffixDelete struct {
	config
	hooks    []Hook
	mutation *WordSuffixMutation
}

// Where appends a list predicates to the WordSuffixDelete builder.
func (wsd *WordSuffixDelete) Where(ps ...predicate.WordSuffix) *WordSuffixDelete {
	wsd.mutation.Where(ps...)
	return wsd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (wsd *WordSuffixDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, wsd.sqlExec, wsd.mutation, wsd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (wsd *WordSuffixDelete) ExecX(ctx context.Context) int {
	n, err := wsd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (wsd *WordSuffixDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(wordsuffix.Table, sqlgraph.NewFieldSpec(wordsuffix.FieldID, field.TypeUUID))
	if ps := wsd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, wsd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	wsd.mutation.done = true
	return affected, err
}

// WordSuffixDeleteOne is the builder for deleting a single WordSuffix entity.
type WordSuffixDeleteOne struct {
	wsd *WordSuffixDelete
}

// Where appends a list predicates to the WordSuffixDelete builder.
func (wsdo *WordSuffixDeleteOne) Where(ps ...predicate.WordSuffix) *WordSuffixDeleteOne {
	wsdo.wsd.mutation.Where(ps...)
	return wsdo
}

// Exec executes the deletion query.
func (wsdo *WordSuffixDeleteOne) Exec(ctx context.Context) error {
	n, err := wsdo.wsd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{wordsuffix.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (wsdo *WordSuffixDeleteOne) ExecX(ctx context.Context) {
	if err := wsdo.Exec(ctx); err != nil {
		panic(err)
	}
}