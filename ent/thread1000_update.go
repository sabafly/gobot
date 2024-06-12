// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/thread1000"
	"github.com/sabafly/gobot/ent/thread1000channel"
)

// Thread1000Update is the builder for updating Thread1000 entities.
type Thread1000Update struct {
	config
	hooks    []Hook
	mutation *Thread1000Mutation
}

// Where appends a list predicates to the Thread1000Update builder.
func (t *Thread1000Update) Where(ps ...predicate.Thread1000) *Thread1000Update {
	t.mutation.Where(ps...)
	return t
}

// SetName sets the "name" field.
func (t *Thread1000Update) SetName(s string) *Thread1000Update {
	t.mutation.SetName(s)
	return t
}

// SetNillableName sets the "name" field if the given value is not nil.
func (t *Thread1000Update) SetNillableName(s *string) *Thread1000Update {
	if s != nil {
		t.SetName(*s)
	}
	return t
}

// SetMessageCount sets the "message_count" field.
func (t *Thread1000Update) SetMessageCount(i int) *Thread1000Update {
	t.mutation.ResetMessageCount()
	t.mutation.SetMessageCount(i)
	return t
}

// SetNillableMessageCount sets the "message_count" field if the given value is not nil.
func (t *Thread1000Update) SetNillableMessageCount(i *int) *Thread1000Update {
	if i != nil {
		t.SetMessageCount(*i)
	}
	return t
}

// AddMessageCount adds i to the "message_count" field.
func (t *Thread1000Update) AddMessageCount(i int) *Thread1000Update {
	t.mutation.AddMessageCount(i)
	return t
}

// SetIsArchived sets the "is_archived" field.
func (t *Thread1000Update) SetIsArchived(b bool) *Thread1000Update {
	t.mutation.SetIsArchived(b)
	return t
}

// SetNillableIsArchived sets the "is_archived" field if the given value is not nil.
func (t *Thread1000Update) SetNillableIsArchived(b *bool) *Thread1000Update {
	if b != nil {
		t.SetIsArchived(*b)
	}
	return t
}

// SetThreadID sets the "thread_id" field.
func (t *Thread1000Update) SetThreadID(s snowflake.ID) *Thread1000Update {
	t.mutation.ResetThreadID()
	t.mutation.SetThreadID(s)
	return t
}

// SetNillableThreadID sets the "thread_id" field if the given value is not nil.
func (t *Thread1000Update) SetNillableThreadID(s *snowflake.ID) *Thread1000Update {
	if s != nil {
		t.SetThreadID(*s)
	}
	return t
}

// AddThreadID adds s to the "thread_id" field.
func (t *Thread1000Update) AddThreadID(s snowflake.ID) *Thread1000Update {
	t.mutation.AddThreadID(s)
	return t
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (t *Thread1000Update) SetGuildID(id snowflake.ID) *Thread1000Update {
	t.mutation.SetGuildID(id)
	return t
}

// SetGuild sets the "guild" edge to the Guild entity.
func (t *Thread1000Update) SetGuild(g *Guild) *Thread1000Update {
	return t.SetGuildID(g.ID)
}

// SetChannelID sets the "channel" edge to the Thread1000Channel entity by ID.
func (t *Thread1000Update) SetChannelID(id uuid.UUID) *Thread1000Update {
	t.mutation.SetChannelID(id)
	return t
}

// SetChannel sets the "channel" edge to the Thread1000Channel entity.
func (t *Thread1000Update) SetChannel(v *Thread1000Channel) *Thread1000Update {
	return t.SetChannelID(v.ID)
}

// Mutation returns the Thread1000Mutation object of the builder.
func (t *Thread1000Update) Mutation() *Thread1000Mutation {
	return t.mutation
}

// ClearGuild clears the "guild" edge to the Guild entity.
func (t *Thread1000Update) ClearGuild() *Thread1000Update {
	t.mutation.ClearGuild()
	return t
}

// ClearChannel clears the "channel" edge to the Thread1000Channel entity.
func (t *Thread1000Update) ClearChannel() *Thread1000Update {
	t.mutation.ClearChannel()
	return t
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (t *Thread1000Update) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, t.sqlSave, t.mutation, t.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (t *Thread1000Update) SaveX(ctx context.Context) int {
	affected, err := t.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (t *Thread1000Update) Exec(ctx context.Context) error {
	_, err := t.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (t *Thread1000Update) ExecX(ctx context.Context) {
	if err := t.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (t *Thread1000Update) check() error {
	if v, ok := t.mutation.Name(); ok {
		if err := thread1000.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "Thread1000.name": %w`, err)}
		}
	}
	if _, ok := t.mutation.GuildID(); t.mutation.GuildCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "Thread1000.guild"`)
	}
	if _, ok := t.mutation.ChannelID(); t.mutation.ChannelCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "Thread1000.channel"`)
	}
	return nil
}

func (t *Thread1000Update) sqlSave(ctx context.Context) (n int, err error) {
	if err := t.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(thread1000.Table, thread1000.Columns, sqlgraph.NewFieldSpec(thread1000.FieldID, field.TypeUUID))
	if ps := t.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := t.mutation.Name(); ok {
		_spec.SetField(thread1000.FieldName, field.TypeString, value)
	}
	if value, ok := t.mutation.MessageCount(); ok {
		_spec.SetField(thread1000.FieldMessageCount, field.TypeInt, value)
	}
	if value, ok := t.mutation.AddedMessageCount(); ok {
		_spec.AddField(thread1000.FieldMessageCount, field.TypeInt, value)
	}
	if value, ok := t.mutation.IsArchived(); ok {
		_spec.SetField(thread1000.FieldIsArchived, field.TypeBool, value)
	}
	if value, ok := t.mutation.ThreadID(); ok {
		_spec.SetField(thread1000.FieldThreadID, field.TypeUint64, value)
	}
	if value, ok := t.mutation.AddedThreadID(); ok {
		_spec.AddField(thread1000.FieldThreadID, field.TypeUint64, value)
	}
	if t.mutation.GuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   thread1000.GuildTable,
			Columns: []string{thread1000.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := t.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   thread1000.GuildTable,
			Columns: []string{thread1000.GuildColumn},
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
	if t.mutation.ChannelCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   thread1000.ChannelTable,
			Columns: []string{thread1000.ChannelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(thread1000channel.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := t.mutation.ChannelIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   thread1000.ChannelTable,
			Columns: []string{thread1000.ChannelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(thread1000channel.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, t.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{thread1000.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	t.mutation.done = true
	return n, nil
}

// Thread1000UpdateOne is the builder for updating a single Thread1000 entity.
type Thread1000UpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *Thread1000Mutation
}

// SetName sets the "name" field.
func (to *Thread1000UpdateOne) SetName(s string) *Thread1000UpdateOne {
	to.mutation.SetName(s)
	return to
}

// SetNillableName sets the "name" field if the given value is not nil.
func (to *Thread1000UpdateOne) SetNillableName(s *string) *Thread1000UpdateOne {
	if s != nil {
		to.SetName(*s)
	}
	return to
}

// SetMessageCount sets the "message_count" field.
func (to *Thread1000UpdateOne) SetMessageCount(i int) *Thread1000UpdateOne {
	to.mutation.ResetMessageCount()
	to.mutation.SetMessageCount(i)
	return to
}

// SetNillableMessageCount sets the "message_count" field if the given value is not nil.
func (to *Thread1000UpdateOne) SetNillableMessageCount(i *int) *Thread1000UpdateOne {
	if i != nil {
		to.SetMessageCount(*i)
	}
	return to
}

// AddMessageCount adds i to the "message_count" field.
func (to *Thread1000UpdateOne) AddMessageCount(i int) *Thread1000UpdateOne {
	to.mutation.AddMessageCount(i)
	return to
}

// SetIsArchived sets the "is_archived" field.
func (to *Thread1000UpdateOne) SetIsArchived(b bool) *Thread1000UpdateOne {
	to.mutation.SetIsArchived(b)
	return to
}

// SetNillableIsArchived sets the "is_archived" field if the given value is not nil.
func (to *Thread1000UpdateOne) SetNillableIsArchived(b *bool) *Thread1000UpdateOne {
	if b != nil {
		to.SetIsArchived(*b)
	}
	return to
}

// SetThreadID sets the "thread_id" field.
func (to *Thread1000UpdateOne) SetThreadID(s snowflake.ID) *Thread1000UpdateOne {
	to.mutation.ResetThreadID()
	to.mutation.SetThreadID(s)
	return to
}

// SetNillableThreadID sets the "thread_id" field if the given value is not nil.
func (to *Thread1000UpdateOne) SetNillableThreadID(s *snowflake.ID) *Thread1000UpdateOne {
	if s != nil {
		to.SetThreadID(*s)
	}
	return to
}

// AddThreadID adds s to the "thread_id" field.
func (to *Thread1000UpdateOne) AddThreadID(s snowflake.ID) *Thread1000UpdateOne {
	to.mutation.AddThreadID(s)
	return to
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (to *Thread1000UpdateOne) SetGuildID(id snowflake.ID) *Thread1000UpdateOne {
	to.mutation.SetGuildID(id)
	return to
}

// SetGuild sets the "guild" edge to the Guild entity.
func (to *Thread1000UpdateOne) SetGuild(g *Guild) *Thread1000UpdateOne {
	return to.SetGuildID(g.ID)
}

// SetChannelID sets the "channel" edge to the Thread1000Channel entity by ID.
func (to *Thread1000UpdateOne) SetChannelID(id uuid.UUID) *Thread1000UpdateOne {
	to.mutation.SetChannelID(id)
	return to
}

// SetChannel sets the "channel" edge to the Thread1000Channel entity.
func (to *Thread1000UpdateOne) SetChannel(t *Thread1000Channel) *Thread1000UpdateOne {
	return to.SetChannelID(t.ID)
}

// Mutation returns the Thread1000Mutation object of the builder.
func (to *Thread1000UpdateOne) Mutation() *Thread1000Mutation {
	return to.mutation
}

// ClearGuild clears the "guild" edge to the Guild entity.
func (to *Thread1000UpdateOne) ClearGuild() *Thread1000UpdateOne {
	to.mutation.ClearGuild()
	return to
}

// ClearChannel clears the "channel" edge to the Thread1000Channel entity.
func (to *Thread1000UpdateOne) ClearChannel() *Thread1000UpdateOne {
	to.mutation.ClearChannel()
	return to
}

// Where appends a list predicates to the Thread1000Update builder.
func (to *Thread1000UpdateOne) Where(ps ...predicate.Thread1000) *Thread1000UpdateOne {
	to.mutation.Where(ps...)
	return to
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (to *Thread1000UpdateOne) Select(field string, fields ...string) *Thread1000UpdateOne {
	to.fields = append([]string{field}, fields...)
	return to
}

// Save executes the query and returns the updated Thread1000 entity.
func (to *Thread1000UpdateOne) Save(ctx context.Context) (*Thread1000, error) {
	return withHooks(ctx, to.sqlSave, to.mutation, to.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (to *Thread1000UpdateOne) SaveX(ctx context.Context) *Thread1000 {
	node, err := to.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (to *Thread1000UpdateOne) Exec(ctx context.Context) error {
	_, err := to.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (to *Thread1000UpdateOne) ExecX(ctx context.Context) {
	if err := to.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (to *Thread1000UpdateOne) check() error {
	if v, ok := to.mutation.Name(); ok {
		if err := thread1000.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "Thread1000.name": %w`, err)}
		}
	}
	if _, ok := to.mutation.GuildID(); to.mutation.GuildCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "Thread1000.guild"`)
	}
	if _, ok := to.mutation.ChannelID(); to.mutation.ChannelCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "Thread1000.channel"`)
	}
	return nil
}

func (to *Thread1000UpdateOne) sqlSave(ctx context.Context) (_node *Thread1000, err error) {
	if err := to.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(thread1000.Table, thread1000.Columns, sqlgraph.NewFieldSpec(thread1000.FieldID, field.TypeUUID))
	id, ok := to.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Thread1000.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := to.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, thread1000.FieldID)
		for _, f := range fields {
			if !thread1000.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != thread1000.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := to.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := to.mutation.Name(); ok {
		_spec.SetField(thread1000.FieldName, field.TypeString, value)
	}
	if value, ok := to.mutation.MessageCount(); ok {
		_spec.SetField(thread1000.FieldMessageCount, field.TypeInt, value)
	}
	if value, ok := to.mutation.AddedMessageCount(); ok {
		_spec.AddField(thread1000.FieldMessageCount, field.TypeInt, value)
	}
	if value, ok := to.mutation.IsArchived(); ok {
		_spec.SetField(thread1000.FieldIsArchived, field.TypeBool, value)
	}
	if value, ok := to.mutation.ThreadID(); ok {
		_spec.SetField(thread1000.FieldThreadID, field.TypeUint64, value)
	}
	if value, ok := to.mutation.AddedThreadID(); ok {
		_spec.AddField(thread1000.FieldThreadID, field.TypeUint64, value)
	}
	if to.mutation.GuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   thread1000.GuildTable,
			Columns: []string{thread1000.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := to.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   thread1000.GuildTable,
			Columns: []string{thread1000.GuildColumn},
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
	if to.mutation.ChannelCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   thread1000.ChannelTable,
			Columns: []string{thread1000.ChannelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(thread1000channel.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := to.mutation.ChannelIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   thread1000.ChannelTable,
			Columns: []string{thread1000.ChannelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(thread1000channel.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Thread1000{config: to.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, to.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{thread1000.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	to.mutation.done = true
	return _node, nil
}
