// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepaneledit"
	"github.com/sabafly/gobot/ent/schema"
)

// RolePanelEditCreate is the builder for creating a RolePanelEdit entity.
type RolePanelEditCreate struct {
	config
	mutation *RolePanelEditMutation
	hooks    []Hook
}

// SetChannelID sets the "channel_id" field.
func (rpec *RolePanelEditCreate) SetChannelID(s snowflake.ID) *RolePanelEditCreate {
	rpec.mutation.SetChannelID(s)
	return rpec
}

// SetEmojiAuthor sets the "emoji_author" field.
func (rpec *RolePanelEditCreate) SetEmojiAuthor(s snowflake.ID) *RolePanelEditCreate {
	rpec.mutation.SetEmojiAuthor(s)
	return rpec
}

// SetNillableEmojiAuthor sets the "emoji_author" field if the given value is not nil.
func (rpec *RolePanelEditCreate) SetNillableEmojiAuthor(s *snowflake.ID) *RolePanelEditCreate {
	if s != nil {
		rpec.SetEmojiAuthor(*s)
	}
	return rpec
}

// SetToken sets the "token" field.
func (rpec *RolePanelEditCreate) SetToken(s string) *RolePanelEditCreate {
	rpec.mutation.SetToken(s)
	return rpec
}

// SetNillableToken sets the "token" field if the given value is not nil.
func (rpec *RolePanelEditCreate) SetNillableToken(s *string) *RolePanelEditCreate {
	if s != nil {
		rpec.SetToken(*s)
	}
	return rpec
}

// SetSelectedRole sets the "selected_role" field.
func (rpec *RolePanelEditCreate) SetSelectedRole(s snowflake.ID) *RolePanelEditCreate {
	rpec.mutation.SetSelectedRole(s)
	return rpec
}

// SetNillableSelectedRole sets the "selected_role" field if the given value is not nil.
func (rpec *RolePanelEditCreate) SetNillableSelectedRole(s *snowflake.ID) *RolePanelEditCreate {
	if s != nil {
		rpec.SetSelectedRole(*s)
	}
	return rpec
}

// SetModified sets the "modified" field.
func (rpec *RolePanelEditCreate) SetModified(b bool) *RolePanelEditCreate {
	rpec.mutation.SetModified(b)
	return rpec
}

// SetNillableModified sets the "modified" field if the given value is not nil.
func (rpec *RolePanelEditCreate) SetNillableModified(b *bool) *RolePanelEditCreate {
	if b != nil {
		rpec.SetModified(*b)
	}
	return rpec
}

// SetName sets the "name" field.
func (rpec *RolePanelEditCreate) SetName(s string) *RolePanelEditCreate {
	rpec.mutation.SetName(s)
	return rpec
}

// SetNillableName sets the "name" field if the given value is not nil.
func (rpec *RolePanelEditCreate) SetNillableName(s *string) *RolePanelEditCreate {
	if s != nil {
		rpec.SetName(*s)
	}
	return rpec
}

// SetDescription sets the "description" field.
func (rpec *RolePanelEditCreate) SetDescription(s string) *RolePanelEditCreate {
	rpec.mutation.SetDescription(s)
	return rpec
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (rpec *RolePanelEditCreate) SetNillableDescription(s *string) *RolePanelEditCreate {
	if s != nil {
		rpec.SetDescription(*s)
	}
	return rpec
}

// SetRoles sets the "roles" field.
func (rpec *RolePanelEditCreate) SetRoles(s []schema.Role) *RolePanelEditCreate {
	rpec.mutation.SetRoles(s)
	return rpec
}

// SetID sets the "id" field.
func (rpec *RolePanelEditCreate) SetID(u uuid.UUID) *RolePanelEditCreate {
	rpec.mutation.SetID(u)
	return rpec
}

// SetNillableID sets the "id" field if the given value is not nil.
func (rpec *RolePanelEditCreate) SetNillableID(u *uuid.UUID) *RolePanelEditCreate {
	if u != nil {
		rpec.SetID(*u)
	}
	return rpec
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (rpec *RolePanelEditCreate) SetGuildID(id snowflake.ID) *RolePanelEditCreate {
	rpec.mutation.SetGuildID(id)
	return rpec
}

// SetGuild sets the "guild" edge to the Guild entity.
func (rpec *RolePanelEditCreate) SetGuild(g *Guild) *RolePanelEditCreate {
	return rpec.SetGuildID(g.ID)
}

// SetParentID sets the "parent" edge to the RolePanel entity by ID.
func (rpec *RolePanelEditCreate) SetParentID(id uuid.UUID) *RolePanelEditCreate {
	rpec.mutation.SetParentID(id)
	return rpec
}

// SetParent sets the "parent" edge to the RolePanel entity.
func (rpec *RolePanelEditCreate) SetParent(r *RolePanel) *RolePanelEditCreate {
	return rpec.SetParentID(r.ID)
}

// Mutation returns the RolePanelEditMutation object of the builder.
func (rpec *RolePanelEditCreate) Mutation() *RolePanelEditMutation {
	return rpec.mutation
}

// Save creates the RolePanelEdit in the database.
func (rpec *RolePanelEditCreate) Save(ctx context.Context) (*RolePanelEdit, error) {
	rpec.defaults()
	return withHooks(ctx, rpec.sqlSave, rpec.mutation, rpec.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (rpec *RolePanelEditCreate) SaveX(ctx context.Context) *RolePanelEdit {
	v, err := rpec.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (rpec *RolePanelEditCreate) Exec(ctx context.Context) error {
	_, err := rpec.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rpec *RolePanelEditCreate) ExecX(ctx context.Context) {
	if err := rpec.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (rpec *RolePanelEditCreate) defaults() {
	if _, ok := rpec.mutation.Modified(); !ok {
		v := rolepaneledit.DefaultModified
		rpec.mutation.SetModified(v)
	}
	if _, ok := rpec.mutation.ID(); !ok {
		v := rolepaneledit.DefaultID()
		rpec.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rpec *RolePanelEditCreate) check() error {
	if _, ok := rpec.mutation.ChannelID(); !ok {
		return &ValidationError{Name: "channel_id", err: errors.New(`ent: missing required field "RolePanelEdit.channel_id"`)}
	}
	if _, ok := rpec.mutation.Modified(); !ok {
		return &ValidationError{Name: "modified", err: errors.New(`ent: missing required field "RolePanelEdit.modified"`)}
	}
	if v, ok := rpec.mutation.Name(); ok {
		if err := rolepaneledit.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "RolePanelEdit.name": %w`, err)}
		}
	}
	if _, ok := rpec.mutation.GuildID(); !ok {
		return &ValidationError{Name: "guild", err: errors.New(`ent: missing required edge "RolePanelEdit.guild"`)}
	}
	if _, ok := rpec.mutation.ParentID(); !ok {
		return &ValidationError{Name: "parent", err: errors.New(`ent: missing required edge "RolePanelEdit.parent"`)}
	}
	return nil
}

func (rpec *RolePanelEditCreate) sqlSave(ctx context.Context) (*RolePanelEdit, error) {
	if err := rpec.check(); err != nil {
		return nil, err
	}
	_node, _spec := rpec.createSpec()
	if err := sqlgraph.CreateNode(ctx, rpec.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(*uuid.UUID); ok {
			_node.ID = *id
		} else if err := _node.ID.Scan(_spec.ID.Value); err != nil {
			return nil, err
		}
	}
	rpec.mutation.id = &_node.ID
	rpec.mutation.done = true
	return _node, nil
}

func (rpec *RolePanelEditCreate) createSpec() (*RolePanelEdit, *sqlgraph.CreateSpec) {
	var (
		_node = &RolePanelEdit{config: rpec.config}
		_spec = sqlgraph.NewCreateSpec(rolepaneledit.Table, sqlgraph.NewFieldSpec(rolepaneledit.FieldID, field.TypeUUID))
	)
	if id, ok := rpec.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := rpec.mutation.ChannelID(); ok {
		_spec.SetField(rolepaneledit.FieldChannelID, field.TypeUint64, value)
		_node.ChannelID = value
	}
	if value, ok := rpec.mutation.EmojiAuthor(); ok {
		_spec.SetField(rolepaneledit.FieldEmojiAuthor, field.TypeUint64, value)
		_node.EmojiAuthor = &value
	}
	if value, ok := rpec.mutation.Token(); ok {
		_spec.SetField(rolepaneledit.FieldToken, field.TypeString, value)
		_node.Token = &value
	}
	if value, ok := rpec.mutation.SelectedRole(); ok {
		_spec.SetField(rolepaneledit.FieldSelectedRole, field.TypeUint64, value)
		_node.SelectedRole = &value
	}
	if value, ok := rpec.mutation.Modified(); ok {
		_spec.SetField(rolepaneledit.FieldModified, field.TypeBool, value)
		_node.Modified = value
	}
	if value, ok := rpec.mutation.Name(); ok {
		_spec.SetField(rolepaneledit.FieldName, field.TypeString, value)
		_node.Name = &value
	}
	if value, ok := rpec.mutation.Description(); ok {
		_spec.SetField(rolepaneledit.FieldDescription, field.TypeString, value)
		_node.Description = &value
	}
	if value, ok := rpec.mutation.Roles(); ok {
		_spec.SetField(rolepaneledit.FieldRoles, field.TypeJSON, value)
		_node.Roles = value
	}
	if nodes := rpec.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   rolepaneledit.GuildTable,
			Columns: []string{rolepaneledit.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.guild_role_panel_edits = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := rpec.mutation.ParentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   rolepaneledit.ParentTable,
			Columns: []string{rolepaneledit.ParentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(rolepanel.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.role_panel_edit = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// RolePanelEditCreateBulk is the builder for creating many RolePanelEdit entities in bulk.
type RolePanelEditCreateBulk struct {
	config
	err      error
	builders []*RolePanelEditCreate
}

// Save creates the RolePanelEdit entities in the database.
func (rpecb *RolePanelEditCreateBulk) Save(ctx context.Context) ([]*RolePanelEdit, error) {
	if rpecb.err != nil {
		return nil, rpecb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(rpecb.builders))
	nodes := make([]*RolePanelEdit, len(rpecb.builders))
	mutators := make([]Mutator, len(rpecb.builders))
	for i := range rpecb.builders {
		func(i int, root context.Context) {
			builder := rpecb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*RolePanelEditMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, rpecb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, rpecb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, rpecb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (rpecb *RolePanelEditCreateBulk) SaveX(ctx context.Context) []*RolePanelEdit {
	v, err := rpecb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (rpecb *RolePanelEditCreateBulk) Exec(ctx context.Context) error {
	_, err := rpecb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rpecb *RolePanelEditCreateBulk) ExecX(ctx context.Context) {
	if err := rpecb.Exec(ctx); err != nil {
		panic(err)
	}
}
