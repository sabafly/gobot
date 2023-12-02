// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/messageremind"
	"github.com/sabafly/gobot/ent/predicate"
)

// MessageRemindUpdate is the builder for updating MessageRemind entities.
type MessageRemindUpdate struct {
	config
	hooks    []Hook
	mutation *MessageRemindMutation
}

// Where appends a list predicates to the MessageRemindUpdate builder.
func (mru *MessageRemindUpdate) Where(ps ...predicate.MessageRemind) *MessageRemindUpdate {
	mru.mutation.Where(ps...)
	return mru
}

// SetChannelID sets the "channel_id" field.
func (mru *MessageRemindUpdate) SetChannelID(s snowflake.ID) *MessageRemindUpdate {
	mru.mutation.ResetChannelID()
	mru.mutation.SetChannelID(s)
	return mru
}

// SetNillableChannelID sets the "channel_id" field if the given value is not nil.
func (mru *MessageRemindUpdate) SetNillableChannelID(s *snowflake.ID) *MessageRemindUpdate {
	if s != nil {
		mru.SetChannelID(*s)
	}
	return mru
}

// AddChannelID adds s to the "channel_id" field.
func (mru *MessageRemindUpdate) AddChannelID(s snowflake.ID) *MessageRemindUpdate {
	mru.mutation.AddChannelID(s)
	return mru
}

// SetAuthorID sets the "author_id" field.
func (mru *MessageRemindUpdate) SetAuthorID(s snowflake.ID) *MessageRemindUpdate {
	mru.mutation.ResetAuthorID()
	mru.mutation.SetAuthorID(s)
	return mru
}

// SetNillableAuthorID sets the "author_id" field if the given value is not nil.
func (mru *MessageRemindUpdate) SetNillableAuthorID(s *snowflake.ID) *MessageRemindUpdate {
	if s != nil {
		mru.SetAuthorID(*s)
	}
	return mru
}

// AddAuthorID adds s to the "author_id" field.
func (mru *MessageRemindUpdate) AddAuthorID(s snowflake.ID) *MessageRemindUpdate {
	mru.mutation.AddAuthorID(s)
	return mru
}

// SetTime sets the "time" field.
func (mru *MessageRemindUpdate) SetTime(t time.Time) *MessageRemindUpdate {
	mru.mutation.SetTime(t)
	return mru
}

// SetNillableTime sets the "time" field if the given value is not nil.
func (mru *MessageRemindUpdate) SetNillableTime(t *time.Time) *MessageRemindUpdate {
	if t != nil {
		mru.SetTime(*t)
	}
	return mru
}

// SetContent sets the "content" field.
func (mru *MessageRemindUpdate) SetContent(s string) *MessageRemindUpdate {
	mru.mutation.SetContent(s)
	return mru
}

// SetNillableContent sets the "content" field if the given value is not nil.
func (mru *MessageRemindUpdate) SetNillableContent(s *string) *MessageRemindUpdate {
	if s != nil {
		mru.SetContent(*s)
	}
	return mru
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (mru *MessageRemindUpdate) SetGuildID(id snowflake.ID) *MessageRemindUpdate {
	mru.mutation.SetGuildID(id)
	return mru
}

// SetGuild sets the "guild" edge to the Guild entity.
func (mru *MessageRemindUpdate) SetGuild(g *Guild) *MessageRemindUpdate {
	return mru.SetGuildID(g.ID)
}

// Mutation returns the MessageRemindMutation object of the builder.
func (mru *MessageRemindUpdate) Mutation() *MessageRemindMutation {
	return mru.mutation
}

// ClearGuild clears the "guild" edge to the Guild entity.
func (mru *MessageRemindUpdate) ClearGuild() *MessageRemindUpdate {
	mru.mutation.ClearGuild()
	return mru
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (mru *MessageRemindUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, mru.sqlSave, mru.mutation, mru.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (mru *MessageRemindUpdate) SaveX(ctx context.Context) int {
	affected, err := mru.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (mru *MessageRemindUpdate) Exec(ctx context.Context) error {
	_, err := mru.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (mru *MessageRemindUpdate) ExecX(ctx context.Context) {
	if err := mru.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (mru *MessageRemindUpdate) check() error {
	if v, ok := mru.mutation.Content(); ok {
		if err := messageremind.ContentValidator(v); err != nil {
			return &ValidationError{Name: "content", err: fmt.Errorf(`ent: validator failed for field "MessageRemind.content": %w`, err)}
		}
	}
	if _, ok := mru.mutation.GuildID(); mru.mutation.GuildCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "MessageRemind.guild"`)
	}
	return nil
}

func (mru *MessageRemindUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := mru.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(messageremind.Table, messageremind.Columns, sqlgraph.NewFieldSpec(messageremind.FieldID, field.TypeUUID))
	if ps := mru.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := mru.mutation.ChannelID(); ok {
		_spec.SetField(messageremind.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := mru.mutation.AddedChannelID(); ok {
		_spec.AddField(messageremind.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := mru.mutation.AuthorID(); ok {
		_spec.SetField(messageremind.FieldAuthorID, field.TypeUint64, value)
	}
	if value, ok := mru.mutation.AddedAuthorID(); ok {
		_spec.AddField(messageremind.FieldAuthorID, field.TypeUint64, value)
	}
	if value, ok := mru.mutation.Time(); ok {
		_spec.SetField(messageremind.FieldTime, field.TypeTime, value)
	}
	if value, ok := mru.mutation.Content(); ok {
		_spec.SetField(messageremind.FieldContent, field.TypeString, value)
	}
	if mru.mutation.GuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   messageremind.GuildTable,
			Columns: []string{messageremind.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := mru.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   messageremind.GuildTable,
			Columns: []string{messageremind.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, mru.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{messageremind.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	mru.mutation.done = true
	return n, nil
}

// MessageRemindUpdateOne is the builder for updating a single MessageRemind entity.
type MessageRemindUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *MessageRemindMutation
}

// SetChannelID sets the "channel_id" field.
func (mruo *MessageRemindUpdateOne) SetChannelID(s snowflake.ID) *MessageRemindUpdateOne {
	mruo.mutation.ResetChannelID()
	mruo.mutation.SetChannelID(s)
	return mruo
}

// SetNillableChannelID sets the "channel_id" field if the given value is not nil.
func (mruo *MessageRemindUpdateOne) SetNillableChannelID(s *snowflake.ID) *MessageRemindUpdateOne {
	if s != nil {
		mruo.SetChannelID(*s)
	}
	return mruo
}

// AddChannelID adds s to the "channel_id" field.
func (mruo *MessageRemindUpdateOne) AddChannelID(s snowflake.ID) *MessageRemindUpdateOne {
	mruo.mutation.AddChannelID(s)
	return mruo
}

// SetAuthorID sets the "author_id" field.
func (mruo *MessageRemindUpdateOne) SetAuthorID(s snowflake.ID) *MessageRemindUpdateOne {
	mruo.mutation.ResetAuthorID()
	mruo.mutation.SetAuthorID(s)
	return mruo
}

// SetNillableAuthorID sets the "author_id" field if the given value is not nil.
func (mruo *MessageRemindUpdateOne) SetNillableAuthorID(s *snowflake.ID) *MessageRemindUpdateOne {
	if s != nil {
		mruo.SetAuthorID(*s)
	}
	return mruo
}

// AddAuthorID adds s to the "author_id" field.
func (mruo *MessageRemindUpdateOne) AddAuthorID(s snowflake.ID) *MessageRemindUpdateOne {
	mruo.mutation.AddAuthorID(s)
	return mruo
}

// SetTime sets the "time" field.
func (mruo *MessageRemindUpdateOne) SetTime(t time.Time) *MessageRemindUpdateOne {
	mruo.mutation.SetTime(t)
	return mruo
}

// SetNillableTime sets the "time" field if the given value is not nil.
func (mruo *MessageRemindUpdateOne) SetNillableTime(t *time.Time) *MessageRemindUpdateOne {
	if t != nil {
		mruo.SetTime(*t)
	}
	return mruo
}

// SetContent sets the "content" field.
func (mruo *MessageRemindUpdateOne) SetContent(s string) *MessageRemindUpdateOne {
	mruo.mutation.SetContent(s)
	return mruo
}

// SetNillableContent sets the "content" field if the given value is not nil.
func (mruo *MessageRemindUpdateOne) SetNillableContent(s *string) *MessageRemindUpdateOne {
	if s != nil {
		mruo.SetContent(*s)
	}
	return mruo
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (mruo *MessageRemindUpdateOne) SetGuildID(id snowflake.ID) *MessageRemindUpdateOne {
	mruo.mutation.SetGuildID(id)
	return mruo
}

// SetGuild sets the "guild" edge to the Guild entity.
func (mruo *MessageRemindUpdateOne) SetGuild(g *Guild) *MessageRemindUpdateOne {
	return mruo.SetGuildID(g.ID)
}

// Mutation returns the MessageRemindMutation object of the builder.
func (mruo *MessageRemindUpdateOne) Mutation() *MessageRemindMutation {
	return mruo.mutation
}

// ClearGuild clears the "guild" edge to the Guild entity.
func (mruo *MessageRemindUpdateOne) ClearGuild() *MessageRemindUpdateOne {
	mruo.mutation.ClearGuild()
	return mruo
}

// Where appends a list predicates to the MessageRemindUpdate builder.
func (mruo *MessageRemindUpdateOne) Where(ps ...predicate.MessageRemind) *MessageRemindUpdateOne {
	mruo.mutation.Where(ps...)
	return mruo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (mruo *MessageRemindUpdateOne) Select(field string, fields ...string) *MessageRemindUpdateOne {
	mruo.fields = append([]string{field}, fields...)
	return mruo
}

// Save executes the query and returns the updated MessageRemind entity.
func (mruo *MessageRemindUpdateOne) Save(ctx context.Context) (*MessageRemind, error) {
	return withHooks(ctx, mruo.sqlSave, mruo.mutation, mruo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (mruo *MessageRemindUpdateOne) SaveX(ctx context.Context) *MessageRemind {
	node, err := mruo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (mruo *MessageRemindUpdateOne) Exec(ctx context.Context) error {
	_, err := mruo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (mruo *MessageRemindUpdateOne) ExecX(ctx context.Context) {
	if err := mruo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (mruo *MessageRemindUpdateOne) check() error {
	if v, ok := mruo.mutation.Content(); ok {
		if err := messageremind.ContentValidator(v); err != nil {
			return &ValidationError{Name: "content", err: fmt.Errorf(`ent: validator failed for field "MessageRemind.content": %w`, err)}
		}
	}
	if _, ok := mruo.mutation.GuildID(); mruo.mutation.GuildCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "MessageRemind.guild"`)
	}
	return nil
}

func (mruo *MessageRemindUpdateOne) sqlSave(ctx context.Context) (_node *MessageRemind, err error) {
	if err := mruo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(messageremind.Table, messageremind.Columns, sqlgraph.NewFieldSpec(messageremind.FieldID, field.TypeUUID))
	id, ok := mruo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "MessageRemind.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := mruo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, messageremind.FieldID)
		for _, f := range fields {
			if !messageremind.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != messageremind.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := mruo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := mruo.mutation.ChannelID(); ok {
		_spec.SetField(messageremind.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := mruo.mutation.AddedChannelID(); ok {
		_spec.AddField(messageremind.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := mruo.mutation.AuthorID(); ok {
		_spec.SetField(messageremind.FieldAuthorID, field.TypeUint64, value)
	}
	if value, ok := mruo.mutation.AddedAuthorID(); ok {
		_spec.AddField(messageremind.FieldAuthorID, field.TypeUint64, value)
	}
	if value, ok := mruo.mutation.Time(); ok {
		_spec.SetField(messageremind.FieldTime, field.TypeTime, value)
	}
	if value, ok := mruo.mutation.Content(); ok {
		_spec.SetField(messageremind.FieldContent, field.TypeString, value)
	}
	if mruo.mutation.GuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   messageremind.GuildTable,
			Columns: []string{messageremind.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := mruo.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   messageremind.GuildTable,
			Columns: []string{messageremind.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &MessageRemind{config: mruo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, mruo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{messageremind.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	mruo.mutation.done = true
	return _node, nil
}
