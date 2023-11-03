// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/user"
)

// GuildCreate is the builder for creating a Guild entity.
type GuildCreate struct {
	config
	mutation *GuildMutation
	hooks    []Hook
}

// SetName sets the "name" field.
func (gc *GuildCreate) SetName(s string) *GuildCreate {
	gc.mutation.SetName(s)
	return gc
}

// SetID sets the "id" field.
func (gc *GuildCreate) SetID(s snowflake.ID) *GuildCreate {
	gc.mutation.SetID(s)
	return gc
}

// SetOwnerID sets the "owner" edge to the User entity by ID.
func (gc *GuildCreate) SetOwnerID(id snowflake.ID) *GuildCreate {
	gc.mutation.SetOwnerID(id)
	return gc
}

// SetOwner sets the "owner" edge to the User entity.
func (gc *GuildCreate) SetOwner(u *User) *GuildCreate {
	return gc.SetOwnerID(u.ID)
}

// AddMemberIDs adds the "members" edge to the User entity by IDs.
func (gc *GuildCreate) AddMemberIDs(ids ...snowflake.ID) *GuildCreate {
	gc.mutation.AddMemberIDs(ids...)
	return gc
}

// AddMembers adds the "members" edges to the User entity.
func (gc *GuildCreate) AddMembers(u ...*User) *GuildCreate {
	ids := make([]snowflake.ID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return gc.AddMemberIDs(ids...)
}

// Mutation returns the GuildMutation object of the builder.
func (gc *GuildCreate) Mutation() *GuildMutation {
	return gc.mutation
}

// Save creates the Guild in the database.
func (gc *GuildCreate) Save(ctx context.Context) (*Guild, error) {
	return withHooks(ctx, gc.sqlSave, gc.mutation, gc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (gc *GuildCreate) SaveX(ctx context.Context) *Guild {
	v, err := gc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (gc *GuildCreate) Exec(ctx context.Context) error {
	_, err := gc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (gc *GuildCreate) ExecX(ctx context.Context) {
	if err := gc.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (gc *GuildCreate) check() error {
	if _, ok := gc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "Guild.name"`)}
	}
	if v, ok := gc.mutation.Name(); ok {
		if err := guild.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "Guild.name": %w`, err)}
		}
	}
	if _, ok := gc.mutation.OwnerID(); !ok {
		return &ValidationError{Name: "owner", err: errors.New(`ent: missing required edge "Guild.owner"`)}
	}
	return nil
}

func (gc *GuildCreate) sqlSave(ctx context.Context) (*Guild, error) {
	if err := gc.check(); err != nil {
		return nil, err
	}
	_node, _spec := gc.createSpec()
	if err := sqlgraph.CreateNode(ctx, gc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != _node.ID {
		id := _spec.ID.Value.(int64)
		_node.ID = snowflake.ID(id)
	}
	gc.mutation.id = &_node.ID
	gc.mutation.done = true
	return _node, nil
}

func (gc *GuildCreate) createSpec() (*Guild, *sqlgraph.CreateSpec) {
	var (
		_node = &Guild{config: gc.config}
		_spec = sqlgraph.NewCreateSpec(guild.Table, sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64))
	)
	if id, ok := gc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := gc.mutation.Name(); ok {
		_spec.SetField(guild.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if nodes := gc.mutation.OwnerIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   guild.OwnerTable,
			Columns: []string{guild.OwnerColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUint64),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.user_own_guilds = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := gc.mutation.MembersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   guild.MembersTable,
			Columns: guild.MembersPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUint64),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// GuildCreateBulk is the builder for creating many Guild entities in bulk.
type GuildCreateBulk struct {
	config
	err      error
	builders []*GuildCreate
}

// Save creates the Guild entities in the database.
func (gcb *GuildCreateBulk) Save(ctx context.Context) ([]*Guild, error) {
	if gcb.err != nil {
		return nil, gcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(gcb.builders))
	nodes := make([]*Guild, len(gcb.builders))
	mutators := make([]Mutator, len(gcb.builders))
	for i := range gcb.builders {
		func(i int, root context.Context) {
			builder := gcb.builders[i]
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*GuildMutation)
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
					_, err = mutators[i+1].Mutate(root, gcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, gcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				if specs[i].ID.Value != nil && nodes[i].ID == 0 {
					id := specs[i].ID.Value.(int64)
					nodes[i].ID = snowflake.ID(id)
				}
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
		if _, err := mutators[0].Mutate(ctx, gcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (gcb *GuildCreateBulk) SaveX(ctx context.Context) []*Guild {
	v, err := gcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (gcb *GuildCreateBulk) Exec(ctx context.Context) error {
	_, err := gcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (gcb *GuildCreateBulk) ExecX(ctx context.Context) {
	if err := gcb.Exec(ctx); err != nil {
		panic(err)
	}
}
