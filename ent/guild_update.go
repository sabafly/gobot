// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/user"
)

// GuildUpdate is the builder for updating Guild entities.
type GuildUpdate struct {
	config
	hooks    []Hook
	mutation *GuildMutation
}

// Where appends a list predicates to the GuildUpdate builder.
func (gu *GuildUpdate) Where(ps ...predicate.Guild) *GuildUpdate {
	gu.mutation.Where(ps...)
	return gu
}

// SetName sets the "name" field.
func (gu *GuildUpdate) SetName(s string) *GuildUpdate {
	gu.mutation.SetName(s)
	return gu
}

// SetLocale sets the "locale" field.
func (gu *GuildUpdate) SetLocale(d discord.Locale) *GuildUpdate {
	gu.mutation.SetLocale(d)
	return gu
}

// SetNillableLocale sets the "locale" field if the given value is not nil.
func (gu *GuildUpdate) SetNillableLocale(d *discord.Locale) *GuildUpdate {
	if d != nil {
		gu.SetLocale(*d)
	}
	return gu
}

// SetOwnerID sets the "owner" edge to the User entity by ID.
func (gu *GuildUpdate) SetOwnerID(id snowflake.ID) *GuildUpdate {
	gu.mutation.SetOwnerID(id)
	return gu
}

// SetOwner sets the "owner" edge to the User entity.
func (gu *GuildUpdate) SetOwner(u *User) *GuildUpdate {
	return gu.SetOwnerID(u.ID)
}

// AddMemberIDs adds the "members" edge to the Member entity by IDs.
func (gu *GuildUpdate) AddMemberIDs(ids ...int) *GuildUpdate {
	gu.mutation.AddMemberIDs(ids...)
	return gu
}

// AddMembers adds the "members" edges to the Member entity.
func (gu *GuildUpdate) AddMembers(m ...*Member) *GuildUpdate {
	ids := make([]int, len(m))
	for i := range m {
		ids[i] = m[i].ID
	}
	return gu.AddMemberIDs(ids...)
}

// Mutation returns the GuildMutation object of the builder.
func (gu *GuildUpdate) Mutation() *GuildMutation {
	return gu.mutation
}

// ClearOwner clears the "owner" edge to the User entity.
func (gu *GuildUpdate) ClearOwner() *GuildUpdate {
	gu.mutation.ClearOwner()
	return gu
}

// ClearMembers clears all "members" edges to the Member entity.
func (gu *GuildUpdate) ClearMembers() *GuildUpdate {
	gu.mutation.ClearMembers()
	return gu
}

// RemoveMemberIDs removes the "members" edge to Member entities by IDs.
func (gu *GuildUpdate) RemoveMemberIDs(ids ...int) *GuildUpdate {
	gu.mutation.RemoveMemberIDs(ids...)
	return gu
}

// RemoveMembers removes "members" edges to Member entities.
func (gu *GuildUpdate) RemoveMembers(m ...*Member) *GuildUpdate {
	ids := make([]int, len(m))
	for i := range m {
		ids[i] = m[i].ID
	}
	return gu.RemoveMemberIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (gu *GuildUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, gu.sqlSave, gu.mutation, gu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (gu *GuildUpdate) SaveX(ctx context.Context) int {
	affected, err := gu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (gu *GuildUpdate) Exec(ctx context.Context) error {
	_, err := gu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (gu *GuildUpdate) ExecX(ctx context.Context) {
	if err := gu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (gu *GuildUpdate) check() error {
	if v, ok := gu.mutation.Name(); ok {
		if err := guild.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "Guild.name": %w`, err)}
		}
	}
	if v, ok := gu.mutation.Locale(); ok {
		if err := guild.LocaleValidator(string(v)); err != nil {
			return &ValidationError{Name: "locale", err: fmt.Errorf(`ent: validator failed for field "Guild.locale": %w`, err)}
		}
	}
	if _, ok := gu.mutation.OwnerID(); gu.mutation.OwnerCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "Guild.owner"`)
	}
	return nil
}

func (gu *GuildUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := gu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(guild.Table, guild.Columns, sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64))
	if ps := gu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := gu.mutation.Name(); ok {
		_spec.SetField(guild.FieldName, field.TypeString, value)
	}
	if value, ok := gu.mutation.Locale(); ok {
		_spec.SetField(guild.FieldLocale, field.TypeString, value)
	}
	if gu.mutation.OwnerCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := gu.mutation.OwnerIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if gu.mutation.MembersCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   guild.MembersTable,
			Columns: guild.MembersPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(member.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := gu.mutation.RemovedMembersIDs(); len(nodes) > 0 && !gu.mutation.MembersCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   guild.MembersTable,
			Columns: guild.MembersPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(member.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := gu.mutation.MembersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   guild.MembersTable,
			Columns: guild.MembersPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(member.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, gu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{guild.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	gu.mutation.done = true
	return n, nil
}

// GuildUpdateOne is the builder for updating a single Guild entity.
type GuildUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *GuildMutation
}

// SetName sets the "name" field.
func (guo *GuildUpdateOne) SetName(s string) *GuildUpdateOne {
	guo.mutation.SetName(s)
	return guo
}

// SetLocale sets the "locale" field.
func (guo *GuildUpdateOne) SetLocale(d discord.Locale) *GuildUpdateOne {
	guo.mutation.SetLocale(d)
	return guo
}

// SetNillableLocale sets the "locale" field if the given value is not nil.
func (guo *GuildUpdateOne) SetNillableLocale(d *discord.Locale) *GuildUpdateOne {
	if d != nil {
		guo.SetLocale(*d)
	}
	return guo
}

// SetOwnerID sets the "owner" edge to the User entity by ID.
func (guo *GuildUpdateOne) SetOwnerID(id snowflake.ID) *GuildUpdateOne {
	guo.mutation.SetOwnerID(id)
	return guo
}

// SetOwner sets the "owner" edge to the User entity.
func (guo *GuildUpdateOne) SetOwner(u *User) *GuildUpdateOne {
	return guo.SetOwnerID(u.ID)
}

// AddMemberIDs adds the "members" edge to the Member entity by IDs.
func (guo *GuildUpdateOne) AddMemberIDs(ids ...int) *GuildUpdateOne {
	guo.mutation.AddMemberIDs(ids...)
	return guo
}

// AddMembers adds the "members" edges to the Member entity.
func (guo *GuildUpdateOne) AddMembers(m ...*Member) *GuildUpdateOne {
	ids := make([]int, len(m))
	for i := range m {
		ids[i] = m[i].ID
	}
	return guo.AddMemberIDs(ids...)
}

// Mutation returns the GuildMutation object of the builder.
func (guo *GuildUpdateOne) Mutation() *GuildMutation {
	return guo.mutation
}

// ClearOwner clears the "owner" edge to the User entity.
func (guo *GuildUpdateOne) ClearOwner() *GuildUpdateOne {
	guo.mutation.ClearOwner()
	return guo
}

// ClearMembers clears all "members" edges to the Member entity.
func (guo *GuildUpdateOne) ClearMembers() *GuildUpdateOne {
	guo.mutation.ClearMembers()
	return guo
}

// RemoveMemberIDs removes the "members" edge to Member entities by IDs.
func (guo *GuildUpdateOne) RemoveMemberIDs(ids ...int) *GuildUpdateOne {
	guo.mutation.RemoveMemberIDs(ids...)
	return guo
}

// RemoveMembers removes "members" edges to Member entities.
func (guo *GuildUpdateOne) RemoveMembers(m ...*Member) *GuildUpdateOne {
	ids := make([]int, len(m))
	for i := range m {
		ids[i] = m[i].ID
	}
	return guo.RemoveMemberIDs(ids...)
}

// Where appends a list predicates to the GuildUpdate builder.
func (guo *GuildUpdateOne) Where(ps ...predicate.Guild) *GuildUpdateOne {
	guo.mutation.Where(ps...)
	return guo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (guo *GuildUpdateOne) Select(field string, fields ...string) *GuildUpdateOne {
	guo.fields = append([]string{field}, fields...)
	return guo
}

// Save executes the query and returns the updated Guild entity.
func (guo *GuildUpdateOne) Save(ctx context.Context) (*Guild, error) {
	return withHooks(ctx, guo.sqlSave, guo.mutation, guo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (guo *GuildUpdateOne) SaveX(ctx context.Context) *Guild {
	node, err := guo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (guo *GuildUpdateOne) Exec(ctx context.Context) error {
	_, err := guo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (guo *GuildUpdateOne) ExecX(ctx context.Context) {
	if err := guo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (guo *GuildUpdateOne) check() error {
	if v, ok := guo.mutation.Name(); ok {
		if err := guild.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "Guild.name": %w`, err)}
		}
	}
	if v, ok := guo.mutation.Locale(); ok {
		if err := guild.LocaleValidator(string(v)); err != nil {
			return &ValidationError{Name: "locale", err: fmt.Errorf(`ent: validator failed for field "Guild.locale": %w`, err)}
		}
	}
	if _, ok := guo.mutation.OwnerID(); guo.mutation.OwnerCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "Guild.owner"`)
	}
	return nil
}

func (guo *GuildUpdateOne) sqlSave(ctx context.Context) (_node *Guild, err error) {
	if err := guo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(guild.Table, guild.Columns, sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64))
	id, ok := guo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Guild.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := guo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, guild.FieldID)
		for _, f := range fields {
			if !guild.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != guild.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := guo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := guo.mutation.Name(); ok {
		_spec.SetField(guild.FieldName, field.TypeString, value)
	}
	if value, ok := guo.mutation.Locale(); ok {
		_spec.SetField(guild.FieldLocale, field.TypeString, value)
	}
	if guo.mutation.OwnerCleared() {
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := guo.mutation.OwnerIDs(); len(nodes) > 0 {
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if guo.mutation.MembersCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   guild.MembersTable,
			Columns: guild.MembersPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(member.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := guo.mutation.RemovedMembersIDs(); len(nodes) > 0 && !guo.mutation.MembersCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   guild.MembersTable,
			Columns: guild.MembersPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(member.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := guo.mutation.MembersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   guild.MembersTable,
			Columns: guild.MembersPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(member.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Guild{config: guo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, guo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{guild.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	guo.mutation.done = true
	return _node, nil
}
