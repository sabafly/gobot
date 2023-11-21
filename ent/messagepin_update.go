// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/dialect/sql/sqljson"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/messagepin"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/schema"
)

// MessagePinUpdate is the builder for updating MessagePin entities.
type MessagePinUpdate struct {
	config
	hooks    []Hook
	mutation *MessagePinMutation
}

// Where appends a list predicates to the MessagePinUpdate builder.
func (mpu *MessagePinUpdate) Where(ps ...predicate.MessagePin) *MessagePinUpdate {
	mpu.mutation.Where(ps...)
	return mpu
}

// SetChannelID sets the "channel_id" field.
func (mpu *MessagePinUpdate) SetChannelID(s snowflake.ID) *MessagePinUpdate {
	mpu.mutation.ResetChannelID()
	mpu.mutation.SetChannelID(s)
	return mpu
}

// SetNillableChannelID sets the "channel_id" field if the given value is not nil.
func (mpu *MessagePinUpdate) SetNillableChannelID(s *snowflake.ID) *MessagePinUpdate {
	if s != nil {
		mpu.SetChannelID(*s)
	}
	return mpu
}

// AddChannelID adds s to the "channel_id" field.
func (mpu *MessagePinUpdate) AddChannelID(s snowflake.ID) *MessagePinUpdate {
	mpu.mutation.AddChannelID(s)
	return mpu
}

// SetContent sets the "content" field.
func (mpu *MessagePinUpdate) SetContent(s string) *MessagePinUpdate {
	mpu.mutation.SetContent(s)
	return mpu
}

// SetNillableContent sets the "content" field if the given value is not nil.
func (mpu *MessagePinUpdate) SetNillableContent(s *string) *MessagePinUpdate {
	if s != nil {
		mpu.SetContent(*s)
	}
	return mpu
}

// ClearContent clears the value of the "content" field.
func (mpu *MessagePinUpdate) ClearContent() *MessagePinUpdate {
	mpu.mutation.ClearContent()
	return mpu
}

// SetEmbeds sets the "embeds" field.
func (mpu *MessagePinUpdate) SetEmbeds(d []discord.Embed) *MessagePinUpdate {
	mpu.mutation.SetEmbeds(d)
	return mpu
}

// AppendEmbeds appends d to the "embeds" field.
func (mpu *MessagePinUpdate) AppendEmbeds(d []discord.Embed) *MessagePinUpdate {
	mpu.mutation.AppendEmbeds(d)
	return mpu
}

// ClearEmbeds clears the value of the "embeds" field.
func (mpu *MessagePinUpdate) ClearEmbeds() *MessagePinUpdate {
	mpu.mutation.ClearEmbeds()
	return mpu
}

// SetBeforeID sets the "before_id" field.
func (mpu *MessagePinUpdate) SetBeforeID(s snowflake.ID) *MessagePinUpdate {
	mpu.mutation.ResetBeforeID()
	mpu.mutation.SetBeforeID(s)
	return mpu
}

// SetNillableBeforeID sets the "before_id" field if the given value is not nil.
func (mpu *MessagePinUpdate) SetNillableBeforeID(s *snowflake.ID) *MessagePinUpdate {
	if s != nil {
		mpu.SetBeforeID(*s)
	}
	return mpu
}

// AddBeforeID adds s to the "before_id" field.
func (mpu *MessagePinUpdate) AddBeforeID(s snowflake.ID) *MessagePinUpdate {
	mpu.mutation.AddBeforeID(s)
	return mpu
}

// ClearBeforeID clears the value of the "before_id" field.
func (mpu *MessagePinUpdate) ClearBeforeID() *MessagePinUpdate {
	mpu.mutation.ClearBeforeID()
	return mpu
}

// SetRateLimit sets the "rate_limit" field.
func (mpu *MessagePinUpdate) SetRateLimit(sl schema.RateLimit) *MessagePinUpdate {
	mpu.mutation.SetRateLimit(sl)
	return mpu
}

// SetNillableRateLimit sets the "rate_limit" field if the given value is not nil.
func (mpu *MessagePinUpdate) SetNillableRateLimit(sl *schema.RateLimit) *MessagePinUpdate {
	if sl != nil {
		mpu.SetRateLimit(*sl)
	}
	return mpu
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (mpu *MessagePinUpdate) SetGuildID(id snowflake.ID) *MessagePinUpdate {
	mpu.mutation.SetGuildID(id)
	return mpu
}

// SetGuild sets the "guild" edge to the Guild entity.
func (mpu *MessagePinUpdate) SetGuild(g *Guild) *MessagePinUpdate {
	return mpu.SetGuildID(g.ID)
}

// Mutation returns the MessagePinMutation object of the builder.
func (mpu *MessagePinUpdate) Mutation() *MessagePinMutation {
	return mpu.mutation
}

// ClearGuild clears the "guild" edge to the Guild entity.
func (mpu *MessagePinUpdate) ClearGuild() *MessagePinUpdate {
	mpu.mutation.ClearGuild()
	return mpu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (mpu *MessagePinUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, mpu.sqlSave, mpu.mutation, mpu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (mpu *MessagePinUpdate) SaveX(ctx context.Context) int {
	affected, err := mpu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (mpu *MessagePinUpdate) Exec(ctx context.Context) error {
	_, err := mpu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (mpu *MessagePinUpdate) ExecX(ctx context.Context) {
	if err := mpu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (mpu *MessagePinUpdate) check() error {
	if _, ok := mpu.mutation.GuildID(); mpu.mutation.GuildCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "MessagePin.guild"`)
	}
	return nil
}

func (mpu *MessagePinUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := mpu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(messagepin.Table, messagepin.Columns, sqlgraph.NewFieldSpec(messagepin.FieldID, field.TypeUUID))
	if ps := mpu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := mpu.mutation.ChannelID(); ok {
		_spec.SetField(messagepin.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := mpu.mutation.AddedChannelID(); ok {
		_spec.AddField(messagepin.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := mpu.mutation.Content(); ok {
		_spec.SetField(messagepin.FieldContent, field.TypeString, value)
	}
	if mpu.mutation.ContentCleared() {
		_spec.ClearField(messagepin.FieldContent, field.TypeString)
	}
	if value, ok := mpu.mutation.Embeds(); ok {
		_spec.SetField(messagepin.FieldEmbeds, field.TypeJSON, value)
	}
	if value, ok := mpu.mutation.AppendedEmbeds(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, messagepin.FieldEmbeds, value)
		})
	}
	if mpu.mutation.EmbedsCleared() {
		_spec.ClearField(messagepin.FieldEmbeds, field.TypeJSON)
	}
	if value, ok := mpu.mutation.BeforeID(); ok {
		_spec.SetField(messagepin.FieldBeforeID, field.TypeUint64, value)
	}
	if value, ok := mpu.mutation.AddedBeforeID(); ok {
		_spec.AddField(messagepin.FieldBeforeID, field.TypeUint64, value)
	}
	if mpu.mutation.BeforeIDCleared() {
		_spec.ClearField(messagepin.FieldBeforeID, field.TypeUint64)
	}
	if value, ok := mpu.mutation.RateLimit(); ok {
		_spec.SetField(messagepin.FieldRateLimit, field.TypeJSON, value)
	}
	if mpu.mutation.GuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   messagepin.GuildTable,
			Columns: []string{messagepin.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := mpu.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   messagepin.GuildTable,
			Columns: []string{messagepin.GuildColumn},
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
	if n, err = sqlgraph.UpdateNodes(ctx, mpu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{messagepin.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	mpu.mutation.done = true
	return n, nil
}

// MessagePinUpdateOne is the builder for updating a single MessagePin entity.
type MessagePinUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *MessagePinMutation
}

// SetChannelID sets the "channel_id" field.
func (mpuo *MessagePinUpdateOne) SetChannelID(s snowflake.ID) *MessagePinUpdateOne {
	mpuo.mutation.ResetChannelID()
	mpuo.mutation.SetChannelID(s)
	return mpuo
}

// SetNillableChannelID sets the "channel_id" field if the given value is not nil.
func (mpuo *MessagePinUpdateOne) SetNillableChannelID(s *snowflake.ID) *MessagePinUpdateOne {
	if s != nil {
		mpuo.SetChannelID(*s)
	}
	return mpuo
}

// AddChannelID adds s to the "channel_id" field.
func (mpuo *MessagePinUpdateOne) AddChannelID(s snowflake.ID) *MessagePinUpdateOne {
	mpuo.mutation.AddChannelID(s)
	return mpuo
}

// SetContent sets the "content" field.
func (mpuo *MessagePinUpdateOne) SetContent(s string) *MessagePinUpdateOne {
	mpuo.mutation.SetContent(s)
	return mpuo
}

// SetNillableContent sets the "content" field if the given value is not nil.
func (mpuo *MessagePinUpdateOne) SetNillableContent(s *string) *MessagePinUpdateOne {
	if s != nil {
		mpuo.SetContent(*s)
	}
	return mpuo
}

// ClearContent clears the value of the "content" field.
func (mpuo *MessagePinUpdateOne) ClearContent() *MessagePinUpdateOne {
	mpuo.mutation.ClearContent()
	return mpuo
}

// SetEmbeds sets the "embeds" field.
func (mpuo *MessagePinUpdateOne) SetEmbeds(d []discord.Embed) *MessagePinUpdateOne {
	mpuo.mutation.SetEmbeds(d)
	return mpuo
}

// AppendEmbeds appends d to the "embeds" field.
func (mpuo *MessagePinUpdateOne) AppendEmbeds(d []discord.Embed) *MessagePinUpdateOne {
	mpuo.mutation.AppendEmbeds(d)
	return mpuo
}

// ClearEmbeds clears the value of the "embeds" field.
func (mpuo *MessagePinUpdateOne) ClearEmbeds() *MessagePinUpdateOne {
	mpuo.mutation.ClearEmbeds()
	return mpuo
}

// SetBeforeID sets the "before_id" field.
func (mpuo *MessagePinUpdateOne) SetBeforeID(s snowflake.ID) *MessagePinUpdateOne {
	mpuo.mutation.ResetBeforeID()
	mpuo.mutation.SetBeforeID(s)
	return mpuo
}

// SetNillableBeforeID sets the "before_id" field if the given value is not nil.
func (mpuo *MessagePinUpdateOne) SetNillableBeforeID(s *snowflake.ID) *MessagePinUpdateOne {
	if s != nil {
		mpuo.SetBeforeID(*s)
	}
	return mpuo
}

// AddBeforeID adds s to the "before_id" field.
func (mpuo *MessagePinUpdateOne) AddBeforeID(s snowflake.ID) *MessagePinUpdateOne {
	mpuo.mutation.AddBeforeID(s)
	return mpuo
}

// ClearBeforeID clears the value of the "before_id" field.
func (mpuo *MessagePinUpdateOne) ClearBeforeID() *MessagePinUpdateOne {
	mpuo.mutation.ClearBeforeID()
	return mpuo
}

// SetRateLimit sets the "rate_limit" field.
func (mpuo *MessagePinUpdateOne) SetRateLimit(sl schema.RateLimit) *MessagePinUpdateOne {
	mpuo.mutation.SetRateLimit(sl)
	return mpuo
}

// SetNillableRateLimit sets the "rate_limit" field if the given value is not nil.
func (mpuo *MessagePinUpdateOne) SetNillableRateLimit(sl *schema.RateLimit) *MessagePinUpdateOne {
	if sl != nil {
		mpuo.SetRateLimit(*sl)
	}
	return mpuo
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (mpuo *MessagePinUpdateOne) SetGuildID(id snowflake.ID) *MessagePinUpdateOne {
	mpuo.mutation.SetGuildID(id)
	return mpuo
}

// SetGuild sets the "guild" edge to the Guild entity.
func (mpuo *MessagePinUpdateOne) SetGuild(g *Guild) *MessagePinUpdateOne {
	return mpuo.SetGuildID(g.ID)
}

// Mutation returns the MessagePinMutation object of the builder.
func (mpuo *MessagePinUpdateOne) Mutation() *MessagePinMutation {
	return mpuo.mutation
}

// ClearGuild clears the "guild" edge to the Guild entity.
func (mpuo *MessagePinUpdateOne) ClearGuild() *MessagePinUpdateOne {
	mpuo.mutation.ClearGuild()
	return mpuo
}

// Where appends a list predicates to the MessagePinUpdate builder.
func (mpuo *MessagePinUpdateOne) Where(ps ...predicate.MessagePin) *MessagePinUpdateOne {
	mpuo.mutation.Where(ps...)
	return mpuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (mpuo *MessagePinUpdateOne) Select(field string, fields ...string) *MessagePinUpdateOne {
	mpuo.fields = append([]string{field}, fields...)
	return mpuo
}

// Save executes the query and returns the updated MessagePin entity.
func (mpuo *MessagePinUpdateOne) Save(ctx context.Context) (*MessagePin, error) {
	return withHooks(ctx, mpuo.sqlSave, mpuo.mutation, mpuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (mpuo *MessagePinUpdateOne) SaveX(ctx context.Context) *MessagePin {
	node, err := mpuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (mpuo *MessagePinUpdateOne) Exec(ctx context.Context) error {
	_, err := mpuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (mpuo *MessagePinUpdateOne) ExecX(ctx context.Context) {
	if err := mpuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (mpuo *MessagePinUpdateOne) check() error {
	if _, ok := mpuo.mutation.GuildID(); mpuo.mutation.GuildCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "MessagePin.guild"`)
	}
	return nil
}

func (mpuo *MessagePinUpdateOne) sqlSave(ctx context.Context) (_node *MessagePin, err error) {
	if err := mpuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(messagepin.Table, messagepin.Columns, sqlgraph.NewFieldSpec(messagepin.FieldID, field.TypeUUID))
	id, ok := mpuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "MessagePin.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := mpuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, messagepin.FieldID)
		for _, f := range fields {
			if !messagepin.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != messagepin.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := mpuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := mpuo.mutation.ChannelID(); ok {
		_spec.SetField(messagepin.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := mpuo.mutation.AddedChannelID(); ok {
		_spec.AddField(messagepin.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := mpuo.mutation.Content(); ok {
		_spec.SetField(messagepin.FieldContent, field.TypeString, value)
	}
	if mpuo.mutation.ContentCleared() {
		_spec.ClearField(messagepin.FieldContent, field.TypeString)
	}
	if value, ok := mpuo.mutation.Embeds(); ok {
		_spec.SetField(messagepin.FieldEmbeds, field.TypeJSON, value)
	}
	if value, ok := mpuo.mutation.AppendedEmbeds(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, messagepin.FieldEmbeds, value)
		})
	}
	if mpuo.mutation.EmbedsCleared() {
		_spec.ClearField(messagepin.FieldEmbeds, field.TypeJSON)
	}
	if value, ok := mpuo.mutation.BeforeID(); ok {
		_spec.SetField(messagepin.FieldBeforeID, field.TypeUint64, value)
	}
	if value, ok := mpuo.mutation.AddedBeforeID(); ok {
		_spec.AddField(messagepin.FieldBeforeID, field.TypeUint64, value)
	}
	if mpuo.mutation.BeforeIDCleared() {
		_spec.ClearField(messagepin.FieldBeforeID, field.TypeUint64)
	}
	if value, ok := mpuo.mutation.RateLimit(); ok {
		_spec.SetField(messagepin.FieldRateLimit, field.TypeJSON, value)
	}
	if mpuo.mutation.GuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   messagepin.GuildTable,
			Columns: []string{messagepin.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := mpuo.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   messagepin.GuildTable,
			Columns: []string{messagepin.GuildColumn},
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
	_node = &MessagePin{config: mpuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, mpuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{messagepin.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	mpuo.mutation.done = true
	return _node, nil
}
