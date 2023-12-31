// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepaneledit"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/schema"
)

// RolePanelCreate is the builder for creating a RolePanel entity.
type RolePanelCreate struct {
	config
	mutation *RolePanelMutation
	hooks    []Hook
}

// SetName sets the "name" field.
func (rpc *RolePanelCreate) SetName(s string) *RolePanelCreate {
	rpc.mutation.SetName(s)
	return rpc
}

// SetDescription sets the "description" field.
func (rpc *RolePanelCreate) SetDescription(s string) *RolePanelCreate {
	rpc.mutation.SetDescription(s)
	return rpc
}

// SetRoles sets the "roles" field.
func (rpc *RolePanelCreate) SetRoles(s []schema.Role) *RolePanelCreate {
	rpc.mutation.SetRoles(s)
	return rpc
}

// SetUpdatedAt sets the "updated_at" field.
func (rpc *RolePanelCreate) SetUpdatedAt(t time.Time) *RolePanelCreate {
	rpc.mutation.SetUpdatedAt(t)
	return rpc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (rpc *RolePanelCreate) SetNillableUpdatedAt(t *time.Time) *RolePanelCreate {
	if t != nil {
		rpc.SetUpdatedAt(*t)
	}
	return rpc
}

// SetAppliedAt sets the "applied_at" field.
func (rpc *RolePanelCreate) SetAppliedAt(t time.Time) *RolePanelCreate {
	rpc.mutation.SetAppliedAt(t)
	return rpc
}

// SetNillableAppliedAt sets the "applied_at" field if the given value is not nil.
func (rpc *RolePanelCreate) SetNillableAppliedAt(t *time.Time) *RolePanelCreate {
	if t != nil {
		rpc.SetAppliedAt(*t)
	}
	return rpc
}

// SetID sets the "id" field.
func (rpc *RolePanelCreate) SetID(u uuid.UUID) *RolePanelCreate {
	rpc.mutation.SetID(u)
	return rpc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (rpc *RolePanelCreate) SetNillableID(u *uuid.UUID) *RolePanelCreate {
	if u != nil {
		rpc.SetID(*u)
	}
	return rpc
}

// SetGuildID sets the "guild" edge to the Guild entity by ID.
func (rpc *RolePanelCreate) SetGuildID(id snowflake.ID) *RolePanelCreate {
	rpc.mutation.SetGuildID(id)
	return rpc
}

// SetGuild sets the "guild" edge to the Guild entity.
func (rpc *RolePanelCreate) SetGuild(g *Guild) *RolePanelCreate {
	return rpc.SetGuildID(g.ID)
}

// AddPlacementIDs adds the "placements" edge to the RolePanelPlaced entity by IDs.
func (rpc *RolePanelCreate) AddPlacementIDs(ids ...uuid.UUID) *RolePanelCreate {
	rpc.mutation.AddPlacementIDs(ids...)
	return rpc
}

// AddPlacements adds the "placements" edges to the RolePanelPlaced entity.
func (rpc *RolePanelCreate) AddPlacements(r ...*RolePanelPlaced) *RolePanelCreate {
	ids := make([]uuid.UUID, len(r))
	for i := range r {
		ids[i] = r[i].ID
	}
	return rpc.AddPlacementIDs(ids...)
}

// SetEditID sets the "edit" edge to the RolePanelEdit entity by ID.
func (rpc *RolePanelCreate) SetEditID(id uuid.UUID) *RolePanelCreate {
	rpc.mutation.SetEditID(id)
	return rpc
}

// SetNillableEditID sets the "edit" edge to the RolePanelEdit entity by ID if the given value is not nil.
func (rpc *RolePanelCreate) SetNillableEditID(id *uuid.UUID) *RolePanelCreate {
	if id != nil {
		rpc = rpc.SetEditID(*id)
	}
	return rpc
}

// SetEdit sets the "edit" edge to the RolePanelEdit entity.
func (rpc *RolePanelCreate) SetEdit(r *RolePanelEdit) *RolePanelCreate {
	return rpc.SetEditID(r.ID)
}

// Mutation returns the RolePanelMutation object of the builder.
func (rpc *RolePanelCreate) Mutation() *RolePanelMutation {
	return rpc.mutation
}

// Save creates the RolePanel in the database.
func (rpc *RolePanelCreate) Save(ctx context.Context) (*RolePanel, error) {
	rpc.defaults()
	return withHooks(ctx, rpc.sqlSave, rpc.mutation, rpc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (rpc *RolePanelCreate) SaveX(ctx context.Context) *RolePanel {
	v, err := rpc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (rpc *RolePanelCreate) Exec(ctx context.Context) error {
	_, err := rpc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rpc *RolePanelCreate) ExecX(ctx context.Context) {
	if err := rpc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (rpc *RolePanelCreate) defaults() {
	if _, ok := rpc.mutation.ID(); !ok {
		v := rolepanel.DefaultID()
		rpc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rpc *RolePanelCreate) check() error {
	if _, ok := rpc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "RolePanel.name"`)}
	}
	if v, ok := rpc.mutation.Name(); ok {
		if err := rolepanel.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "RolePanel.name": %w`, err)}
		}
	}
	if _, ok := rpc.mutation.Description(); !ok {
		return &ValidationError{Name: "description", err: errors.New(`ent: missing required field "RolePanel.description"`)}
	}
	if v, ok := rpc.mutation.Description(); ok {
		if err := rolepanel.DescriptionValidator(v); err != nil {
			return &ValidationError{Name: "description", err: fmt.Errorf(`ent: validator failed for field "RolePanel.description": %w`, err)}
		}
	}
	if _, ok := rpc.mutation.GuildID(); !ok {
		return &ValidationError{Name: "guild", err: errors.New(`ent: missing required edge "RolePanel.guild"`)}
	}
	return nil
}

func (rpc *RolePanelCreate) sqlSave(ctx context.Context) (*RolePanel, error) {
	if err := rpc.check(); err != nil {
		return nil, err
	}
	_node, _spec := rpc.createSpec()
	if err := sqlgraph.CreateNode(ctx, rpc.driver, _spec); err != nil {
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
	rpc.mutation.id = &_node.ID
	rpc.mutation.done = true
	return _node, nil
}

func (rpc *RolePanelCreate) createSpec() (*RolePanel, *sqlgraph.CreateSpec) {
	var (
		_node = &RolePanel{config: rpc.config}
		_spec = sqlgraph.NewCreateSpec(rolepanel.Table, sqlgraph.NewFieldSpec(rolepanel.FieldID, field.TypeUUID))
	)
	if id, ok := rpc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := rpc.mutation.Name(); ok {
		_spec.SetField(rolepanel.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := rpc.mutation.Description(); ok {
		_spec.SetField(rolepanel.FieldDescription, field.TypeString, value)
		_node.Description = value
	}
	if value, ok := rpc.mutation.Roles(); ok {
		_spec.SetField(rolepanel.FieldRoles, field.TypeJSON, value)
		_node.Roles = value
	}
	if value, ok := rpc.mutation.UpdatedAt(); ok {
		_spec.SetField(rolepanel.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := rpc.mutation.AppliedAt(); ok {
		_spec.SetField(rolepanel.FieldAppliedAt, field.TypeTime, value)
		_node.AppliedAt = value
	}
	if nodes := rpc.mutation.GuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   rolepanel.GuildTable,
			Columns: []string{rolepanel.GuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.guild_role_panels = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := rpc.mutation.PlacementsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   rolepanel.PlacementsTable,
			Columns: []string{rolepanel.PlacementsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(rolepanelplaced.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := rpc.mutation.EditIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   rolepanel.EditTable,
			Columns: []string{rolepanel.EditColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(rolepaneledit.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// RolePanelCreateBulk is the builder for creating many RolePanel entities in bulk.
type RolePanelCreateBulk struct {
	config
	err      error
	builders []*RolePanelCreate
}

// Save creates the RolePanel entities in the database.
func (rpcb *RolePanelCreateBulk) Save(ctx context.Context) ([]*RolePanel, error) {
	if rpcb.err != nil {
		return nil, rpcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(rpcb.builders))
	nodes := make([]*RolePanel, len(rpcb.builders))
	mutators := make([]Mutator, len(rpcb.builders))
	for i := range rpcb.builders {
		func(i int, root context.Context) {
			builder := rpcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*RolePanelMutation)
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
					_, err = mutators[i+1].Mutate(root, rpcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, rpcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, rpcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (rpcb *RolePanelCreateBulk) SaveX(ctx context.Context) []*RolePanel {
	v, err := rpcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (rpcb *RolePanelCreateBulk) Exec(ctx context.Context) error {
	_, err := rpcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rpcb *RolePanelCreateBulk) ExecX(ctx context.Context) {
	if err := rpcb.Exec(ctx); err != nil {
		panic(err)
	}
}
