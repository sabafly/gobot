// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/schema"
)

// RolePanelPlacedCreate is the builder for creating a RolePanelPlaced entity.
type RolePanelPlacedCreate struct {
	config
	mutation *RolePanelPlacedMutation
	hooks    []Hook
}

// SetMessageID sets the "message_id" field.
func (rppc *RolePanelPlacedCreate) SetMessageID(s snowflake.ID) *RolePanelPlacedCreate {
	rppc.mutation.SetMessageID(s)
	return rppc
}

// SetNillableMessageID sets the "message_id" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableMessageID(s *snowflake.ID) *RolePanelPlacedCreate {
	if s != nil {
		rppc.SetMessageID(*s)
	}
	return rppc
}

// SetChannelID sets the "channel_id" field.
func (rppc *RolePanelPlacedCreate) SetChannelID(s snowflake.ID) *RolePanelPlacedCreate {
	rppc.mutation.SetChannelID(s)
	return rppc
}

// SetType sets the "type" field.
func (rppc *RolePanelPlacedCreate) SetType(r rolepanelplaced.Type) *RolePanelPlacedCreate {
	rppc.mutation.SetType(r)
	return rppc
}

// SetNillableType sets the "type" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableType(r *rolepanelplaced.Type) *RolePanelPlacedCreate {
	if r != nil {
		rppc.SetType(*r)
	}
	return rppc
}

// SetButtonType sets the "button_type" field.
func (rppc *RolePanelPlacedCreate) SetButtonType(ds discord.ButtonStyle) *RolePanelPlacedCreate {
	rppc.mutation.SetButtonType(ds)
	return rppc
}

// SetNillableButtonType sets the "button_type" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableButtonType(ds *discord.ButtonStyle) *RolePanelPlacedCreate {
	if ds != nil {
		rppc.SetButtonType(*ds)
	}
	return rppc
}

// SetShowName sets the "show_name" field.
func (rppc *RolePanelPlacedCreate) SetShowName(b bool) *RolePanelPlacedCreate {
	rppc.mutation.SetShowName(b)
	return rppc
}

// SetNillableShowName sets the "show_name" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableShowName(b *bool) *RolePanelPlacedCreate {
	if b != nil {
		rppc.SetShowName(*b)
	}
	return rppc
}

// SetFoldingSelectMenu sets the "folding_select_menu" field.
func (rppc *RolePanelPlacedCreate) SetFoldingSelectMenu(b bool) *RolePanelPlacedCreate {
	rppc.mutation.SetFoldingSelectMenu(b)
	return rppc
}

// SetNillableFoldingSelectMenu sets the "folding_select_menu" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableFoldingSelectMenu(b *bool) *RolePanelPlacedCreate {
	if b != nil {
		rppc.SetFoldingSelectMenu(*b)
	}
	return rppc
}

// SetHideNotice sets the "hide_notice" field.
func (rppc *RolePanelPlacedCreate) SetHideNotice(b bool) *RolePanelPlacedCreate {
	rppc.mutation.SetHideNotice(b)
	return rppc
}

// SetNillableHideNotice sets the "hide_notice" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableHideNotice(b *bool) *RolePanelPlacedCreate {
	if b != nil {
		rppc.SetHideNotice(*b)
	}
	return rppc
}

// SetUseDisplayName sets the "use_display_name" field.
func (rppc *RolePanelPlacedCreate) SetUseDisplayName(b bool) *RolePanelPlacedCreate {
	rppc.mutation.SetUseDisplayName(b)
	return rppc
}

// SetNillableUseDisplayName sets the "use_display_name" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableUseDisplayName(b *bool) *RolePanelPlacedCreate {
	if b != nil {
		rppc.SetUseDisplayName(*b)
	}
	return rppc
}

// SetCreatedAt sets the "created_at" field.
func (rppc *RolePanelPlacedCreate) SetCreatedAt(t time.Time) *RolePanelPlacedCreate {
	rppc.mutation.SetCreatedAt(t)
	return rppc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableCreatedAt(t *time.Time) *RolePanelPlacedCreate {
	if t != nil {
		rppc.SetCreatedAt(*t)
	}
	return rppc
}

// SetUses sets the "uses" field.
func (rppc *RolePanelPlacedCreate) SetUses(i int) *RolePanelPlacedCreate {
	rppc.mutation.SetUses(i)
	return rppc
}

// SetNillableUses sets the "uses" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableUses(i *int) *RolePanelPlacedCreate {
	if i != nil {
		rppc.SetUses(*i)
	}
	return rppc
}

// SetName sets the "name" field.
func (rppc *RolePanelPlacedCreate) SetName(s string) *RolePanelPlacedCreate {
	rppc.mutation.SetName(s)
	return rppc
}

// SetDescription sets the "description" field.
func (rppc *RolePanelPlacedCreate) SetDescription(s string) *RolePanelPlacedCreate {
	rppc.mutation.SetDescription(s)
	return rppc
}

// SetRoles sets the "roles" field.
func (rppc *RolePanelPlacedCreate) SetRoles(s []schema.Role) *RolePanelPlacedCreate {
	rppc.mutation.SetRoles(s)
	return rppc
}

// SetUpdatedAt sets the "updated_at" field.
func (rppc *RolePanelPlacedCreate) SetUpdatedAt(t time.Time) *RolePanelPlacedCreate {
	rppc.mutation.SetUpdatedAt(t)
	return rppc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableUpdatedAt(t *time.Time) *RolePanelPlacedCreate {
	if t != nil {
		rppc.SetUpdatedAt(*t)
	}
	return rppc
}

// SetID sets the "id" field.
func (rppc *RolePanelPlacedCreate) SetID(u uuid.UUID) *RolePanelPlacedCreate {
	rppc.mutation.SetID(u)
	return rppc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (rppc *RolePanelPlacedCreate) SetNillableID(u *uuid.UUID) *RolePanelPlacedCreate {
	if u != nil {
		rppc.SetID(*u)
	}
	return rppc
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (rppc *RolePanelPlacedCreate) SetGuildID(id snowflake.ID) *RolePanelPlacedCreate {
	rppc.mutation.SetGuildID(id)
	return rppc
}

// SetGuild sets the "guild" edge to the Guild entity.
func (rppc *RolePanelPlacedCreate) SetGuild(g *Guild) *RolePanelPlacedCreate {
	return rppc.SetGuildID(g.ID)
}

// SetRolePanelID sets the "role_panel" edge to the RolePanel entity by ID.
func (rppc *RolePanelPlacedCreate) SetRolePanelID(id uuid.UUID) *RolePanelPlacedCreate {
	rppc.mutation.SetRolePanelID(id)
	return rppc
}

// SetRolePanel sets the "role_panel" edge to the RolePanel entity.
func (rppc *RolePanelPlacedCreate) SetRolePanel(r *RolePanel) *RolePanelPlacedCreate {
	return rppc.SetRolePanelID(r.ID)
}

// Mutation returns the RolePanelPlacedMutation object of the builder.
func (rppc *RolePanelPlacedCreate) Mutation() *RolePanelPlacedMutation {
	return rppc.mutation
}

// Save creates the RolePanelPlaced in the database.
func (rppc *RolePanelPlacedCreate) Save(ctx context.Context) (*RolePanelPlaced, error) {
	rppc.defaults()
	return withHooks(ctx, rppc.sqlSave, rppc.mutation, rppc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (rppc *RolePanelPlacedCreate) SaveX(ctx context.Context) *RolePanelPlaced {
	v, err := rppc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (rppc *RolePanelPlacedCreate) Exec(ctx context.Context) error {
	_, err := rppc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rppc *RolePanelPlacedCreate) ExecX(ctx context.Context) {
	if err := rppc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (rppc *RolePanelPlacedCreate) defaults() {
	if _, ok := rppc.mutation.ButtonType(); !ok {
		v := rolepanelplaced.DefaultButtonType
		rppc.mutation.SetButtonType(v)
	}
	if _, ok := rppc.mutation.ShowName(); !ok {
		v := rolepanelplaced.DefaultShowName
		rppc.mutation.SetShowName(v)
	}
	if _, ok := rppc.mutation.FoldingSelectMenu(); !ok {
		v := rolepanelplaced.DefaultFoldingSelectMenu
		rppc.mutation.SetFoldingSelectMenu(v)
	}
	if _, ok := rppc.mutation.HideNotice(); !ok {
		v := rolepanelplaced.DefaultHideNotice
		rppc.mutation.SetHideNotice(v)
	}
	if _, ok := rppc.mutation.UseDisplayName(); !ok {
		v := rolepanelplaced.DefaultUseDisplayName
		rppc.mutation.SetUseDisplayName(v)
	}
	if _, ok := rppc.mutation.CreatedAt(); !ok {
		v := rolepanelplaced.DefaultCreatedAt()
		rppc.mutation.SetCreatedAt(v)
	}
	if _, ok := rppc.mutation.Uses(); !ok {
		v := rolepanelplaced.DefaultUses
		rppc.mutation.SetUses(v)
	}
	if _, ok := rppc.mutation.ID(); !ok {
		v := rolepanelplaced.DefaultID()
		rppc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rppc *RolePanelPlacedCreate) check() error {
	if _, ok := rppc.mutation.ChannelID(); !ok {
		return &ValidationError{Name: "channel_id", err: errors.New(`ent: missing required field "RolePanelPlaced.channel_id"`)}
	}
	if v, ok := rppc.mutation.GetType(); ok {
		if err := rolepanelplaced.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`ent: validator failed for field "RolePanelPlaced.type": %w`, err)}
		}
	}
	if _, ok := rppc.mutation.ButtonType(); !ok {
		return &ValidationError{Name: "button_type", err: errors.New(`ent: missing required field "RolePanelPlaced.button_type"`)}
	}
	if v, ok := rppc.mutation.ButtonType(); ok {
		if err := rolepanelplaced.ButtonTypeValidator(int(v)); err != nil {
			return &ValidationError{Name: "button_type", err: fmt.Errorf(`ent: validator failed for field "RolePanelPlaced.button_type": %w`, err)}
		}
	}
	if _, ok := rppc.mutation.ShowName(); !ok {
		return &ValidationError{Name: "show_name", err: errors.New(`ent: missing required field "RolePanelPlaced.show_name"`)}
	}
	if _, ok := rppc.mutation.FoldingSelectMenu(); !ok {
		return &ValidationError{Name: "folding_select_menu", err: errors.New(`ent: missing required field "RolePanelPlaced.folding_select_menu"`)}
	}
	if _, ok := rppc.mutation.HideNotice(); !ok {
		return &ValidationError{Name: "hide_notice", err: errors.New(`ent: missing required field "RolePanelPlaced.hide_notice"`)}
	}
	if _, ok := rppc.mutation.UseDisplayName(); !ok {
		return &ValidationError{Name: "use_display_name", err: errors.New(`ent: missing required field "RolePanelPlaced.use_display_name"`)}
	}
	if _, ok := rppc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`ent: missing required field "RolePanelPlaced.created_at"`)}
	}
	if _, ok := rppc.mutation.Uses(); !ok {
		return &ValidationError{Name: "uses", err: errors.New(`ent: missing required field "RolePanelPlaced.uses"`)}
	}
	if _, ok := rppc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "RolePanelPlaced.name"`)}
	}
	if v, ok := rppc.mutation.Name(); ok {
		if err := rolepanelplaced.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "RolePanelPlaced.name": %w`, err)}
		}
	}
	if _, ok := rppc.mutation.Description(); !ok {
		return &ValidationError{Name: "description", err: errors.New(`ent: missing required field "RolePanelPlaced.description"`)}
	}
	if _, ok := rppc.mutation.GuildID(); !ok {
		return &ValidationError{Name: "guild", err: errors.New(`ent: missing required edge "RolePanelPlaced.guild"`)}
	}
	if _, ok := rppc.mutation.RolePanelID(); !ok {
		return &ValidationError{Name: "role_panel", err: errors.New(`ent: missing required edge "RolePanelPlaced.role_panel"`)}
	}
	return nil
}

func (rppc *RolePanelPlacedCreate) sqlSave(ctx context.Context) (*RolePanelPlaced, error) {
	if err := rppc.check(); err != nil {
		return nil, err
	}
	_node, _spec := rppc.createSpec()
	if err := sqlgraph.CreateNode(ctx, rppc.driver, _spec); err != nil {
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
	rppc.mutation.id = &_node.ID
	rppc.mutation.done = true
	return _node, nil
}

func (rppc *RolePanelPlacedCreate) createSpec() (*RolePanelPlaced, *sqlgraph.CreateSpec) {
	var (
		_node = &RolePanelPlaced{config: rppc.config}
		_spec = sqlgraph.NewCreateSpec(rolepanelplaced.Table, sqlgraph.NewFieldSpec(rolepanelplaced.FieldID, field.TypeUUID))
	)
	if id, ok := rppc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := rppc.mutation.MessageID(); ok {
		_spec.SetField(rolepanelplaced.FieldMessageID, field.TypeUint64, value)
		_node.MessageID = &value
	}
	if value, ok := rppc.mutation.ChannelID(); ok {
		_spec.SetField(rolepanelplaced.FieldChannelID, field.TypeUint64, value)
		_node.ChannelID = value
	}
	if value, ok := rppc.mutation.GetType(); ok {
		_spec.SetField(rolepanelplaced.FieldType, field.TypeEnum, value)
		_node.Type = value
	}
	if value, ok := rppc.mutation.ButtonType(); ok {
		_spec.SetField(rolepanelplaced.FieldButtonType, field.TypeInt, value)
		_node.ButtonType = value
	}
	if value, ok := rppc.mutation.ShowName(); ok {
		_spec.SetField(rolepanelplaced.FieldShowName, field.TypeBool, value)
		_node.ShowName = value
	}
	if value, ok := rppc.mutation.FoldingSelectMenu(); ok {
		_spec.SetField(rolepanelplaced.FieldFoldingSelectMenu, field.TypeBool, value)
		_node.FoldingSelectMenu = value
	}
	if value, ok := rppc.mutation.HideNotice(); ok {
		_spec.SetField(rolepanelplaced.FieldHideNotice, field.TypeBool, value)
		_node.HideNotice = value
	}
	if value, ok := rppc.mutation.UseDisplayName(); ok {
		_spec.SetField(rolepanelplaced.FieldUseDisplayName, field.TypeBool, value)
		_node.UseDisplayName = value
	}
	if value, ok := rppc.mutation.CreatedAt(); ok {
		_spec.SetField(rolepanelplaced.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := rppc.mutation.Uses(); ok {
		_spec.SetField(rolepanelplaced.FieldUses, field.TypeInt, value)
		_node.Uses = value
	}
	if value, ok := rppc.mutation.Name(); ok {
		_spec.SetField(rolepanelplaced.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := rppc.mutation.Description(); ok {
		_spec.SetField(rolepanelplaced.FieldDescription, field.TypeString, value)
		_node.Description = value
	}
	if value, ok := rppc.mutation.Roles(); ok {
		_spec.SetField(rolepanelplaced.FieldRoles, field.TypeJSON, value)
		_node.Roles = value
	}
	if value, ok := rppc.mutation.UpdatedAt(); ok {
		_spec.SetField(rolepanelplaced.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if nodes := rppc.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   rolepanelplaced.GuildTable,
			Columns: []string{rolepanelplaced.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.guild_role_panel_placements = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := rppc.mutation.RolePanelIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   rolepanelplaced.RolePanelTable,
			Columns: []string{rolepanelplaced.RolePanelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(rolepanel.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.role_panel_placements = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// RolePanelPlacedCreateBulk is the builder for creating many RolePanelPlaced entities in bulk.
type RolePanelPlacedCreateBulk struct {
	config
	err      error
	builders []*RolePanelPlacedCreate
}

// Save creates the RolePanelPlaced entities in the database.
func (rppcb *RolePanelPlacedCreateBulk) Save(ctx context.Context) ([]*RolePanelPlaced, error) {
	if rppcb.err != nil {
		return nil, rppcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(rppcb.builders))
	nodes := make([]*RolePanelPlaced, len(rppcb.builders))
	mutators := make([]Mutator, len(rppcb.builders))
	for i := range rppcb.builders {
		func(i int, root context.Context) {
			builder := rppcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*RolePanelPlacedMutation)
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
					_, err = mutators[i+1].Mutate(root, rppcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, rppcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, rppcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (rppcb *RolePanelPlacedCreateBulk) SaveX(ctx context.Context) []*RolePanelPlaced {
	v, err := rppcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (rppcb *RolePanelPlacedCreateBulk) Exec(ctx context.Context) error {
	_, err := rppcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rppcb *RolePanelPlacedCreateBulk) ExecX(ctx context.Context) {
	if err := rppcb.Exec(ctx); err != nil {
		panic(err)
	}
}
