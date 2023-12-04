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
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/rolepaneledit"
)

// RolePanelEditUpdate is the builder for updating RolePanelEdit entities.
type RolePanelEditUpdate struct {
	config
	hooks    []Hook
	mutation *RolePanelEditMutation
}

// Where appends a list predicates to the RolePanelEditUpdate builder.
func (rpeu *RolePanelEditUpdate) Where(ps ...predicate.RolePanelEdit) *RolePanelEditUpdate {
	rpeu.mutation.Where(ps...)
	return rpeu
}

// SetChannelID sets the "channel_id" field.
func (rpeu *RolePanelEditUpdate) SetChannelID(s snowflake.ID) *RolePanelEditUpdate {
	rpeu.mutation.ResetChannelID()
	rpeu.mutation.SetChannelID(s)
	return rpeu
}

// SetNillableChannelID sets the "channel_id" field if the given value is not nil.
func (rpeu *RolePanelEditUpdate) SetNillableChannelID(s *snowflake.ID) *RolePanelEditUpdate {
	if s != nil {
		rpeu.SetChannelID(*s)
	}
	return rpeu
}

// AddChannelID adds s to the "channel_id" field.
func (rpeu *RolePanelEditUpdate) AddChannelID(s snowflake.ID) *RolePanelEditUpdate {
	rpeu.mutation.AddChannelID(s)
	return rpeu
}

// SetEmojiAuthor sets the "emoji_author" field.
func (rpeu *RolePanelEditUpdate) SetEmojiAuthor(s snowflake.ID) *RolePanelEditUpdate {
	rpeu.mutation.ResetEmojiAuthor()
	rpeu.mutation.SetEmojiAuthor(s)
	return rpeu
}

// SetNillableEmojiAuthor sets the "emoji_author" field if the given value is not nil.
func (rpeu *RolePanelEditUpdate) SetNillableEmojiAuthor(s *snowflake.ID) *RolePanelEditUpdate {
	if s != nil {
		rpeu.SetEmojiAuthor(*s)
	}
	return rpeu
}

// AddEmojiAuthor adds s to the "emoji_author" field.
func (rpeu *RolePanelEditUpdate) AddEmojiAuthor(s snowflake.ID) *RolePanelEditUpdate {
	rpeu.mutation.AddEmojiAuthor(s)
	return rpeu
}

// ClearEmojiAuthor clears the value of the "emoji_author" field.
func (rpeu *RolePanelEditUpdate) ClearEmojiAuthor() *RolePanelEditUpdate {
	rpeu.mutation.ClearEmojiAuthor()
	return rpeu
}

// SetToken sets the "token" field.
func (rpeu *RolePanelEditUpdate) SetToken(s string) *RolePanelEditUpdate {
	rpeu.mutation.SetToken(s)
	return rpeu
}

// SetNillableToken sets the "token" field if the given value is not nil.
func (rpeu *RolePanelEditUpdate) SetNillableToken(s *string) *RolePanelEditUpdate {
	if s != nil {
		rpeu.SetToken(*s)
	}
	return rpeu
}

// ClearToken clears the value of the "token" field.
func (rpeu *RolePanelEditUpdate) ClearToken() *RolePanelEditUpdate {
	rpeu.mutation.ClearToken()
	return rpeu
}

// SetSelectedRole sets the "selected_role" field.
func (rpeu *RolePanelEditUpdate) SetSelectedRole(s snowflake.ID) *RolePanelEditUpdate {
	rpeu.mutation.ResetSelectedRole()
	rpeu.mutation.SetSelectedRole(s)
	return rpeu
}

// SetNillableSelectedRole sets the "selected_role" field if the given value is not nil.
func (rpeu *RolePanelEditUpdate) SetNillableSelectedRole(s *snowflake.ID) *RolePanelEditUpdate {
	if s != nil {
		rpeu.SetSelectedRole(*s)
	}
	return rpeu
}

// AddSelectedRole adds s to the "selected_role" field.
func (rpeu *RolePanelEditUpdate) AddSelectedRole(s snowflake.ID) *RolePanelEditUpdate {
	rpeu.mutation.AddSelectedRole(s)
	return rpeu
}

// ClearSelectedRole clears the value of the "selected_role" field.
func (rpeu *RolePanelEditUpdate) ClearSelectedRole() *RolePanelEditUpdate {
	rpeu.mutation.ClearSelectedRole()
	return rpeu
}

// SetModified sets the "modified" field.
func (rpeu *RolePanelEditUpdate) SetModified(b bool) *RolePanelEditUpdate {
	rpeu.mutation.SetModified(b)
	return rpeu
}

// SetNillableModified sets the "modified" field if the given value is not nil.
func (rpeu *RolePanelEditUpdate) SetNillableModified(b *bool) *RolePanelEditUpdate {
	if b != nil {
		rpeu.SetModified(*b)
	}
	return rpeu
}

// Mutation returns the RolePanelEditMutation object of the builder.
func (rpeu *RolePanelEditUpdate) Mutation() *RolePanelEditMutation {
	return rpeu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (rpeu *RolePanelEditUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, rpeu.sqlSave, rpeu.mutation, rpeu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (rpeu *RolePanelEditUpdate) SaveX(ctx context.Context) int {
	affected, err := rpeu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (rpeu *RolePanelEditUpdate) Exec(ctx context.Context) error {
	_, err := rpeu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rpeu *RolePanelEditUpdate) ExecX(ctx context.Context) {
	if err := rpeu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rpeu *RolePanelEditUpdate) check() error {
	if _, ok := rpeu.mutation.GuildID(); rpeu.mutation.GuildCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "RolePanelEdit.guild"`)
	}
	if _, ok := rpeu.mutation.ParentID(); rpeu.mutation.ParentCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "RolePanelEdit.parent"`)
	}
	return nil
}

func (rpeu *RolePanelEditUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := rpeu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(rolepaneledit.Table, rolepaneledit.Columns, sqlgraph.NewFieldSpec(rolepaneledit.FieldID, field.TypeUUID))
	if ps := rpeu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := rpeu.mutation.ChannelID(); ok {
		_spec.SetField(rolepaneledit.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := rpeu.mutation.AddedChannelID(); ok {
		_spec.AddField(rolepaneledit.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := rpeu.mutation.EmojiAuthor(); ok {
		_spec.SetField(rolepaneledit.FieldEmojiAuthor, field.TypeUint64, value)
	}
	if value, ok := rpeu.mutation.AddedEmojiAuthor(); ok {
		_spec.AddField(rolepaneledit.FieldEmojiAuthor, field.TypeUint64, value)
	}
	if rpeu.mutation.EmojiAuthorCleared() {
		_spec.ClearField(rolepaneledit.FieldEmojiAuthor, field.TypeUint64)
	}
	if value, ok := rpeu.mutation.Token(); ok {
		_spec.SetField(rolepaneledit.FieldToken, field.TypeString, value)
	}
	if rpeu.mutation.TokenCleared() {
		_spec.ClearField(rolepaneledit.FieldToken, field.TypeString)
	}
	if value, ok := rpeu.mutation.SelectedRole(); ok {
		_spec.SetField(rolepaneledit.FieldSelectedRole, field.TypeUint64, value)
	}
	if value, ok := rpeu.mutation.AddedSelectedRole(); ok {
		_spec.AddField(rolepaneledit.FieldSelectedRole, field.TypeUint64, value)
	}
	if rpeu.mutation.SelectedRoleCleared() {
		_spec.ClearField(rolepaneledit.FieldSelectedRole, field.TypeUint64)
	}
	if value, ok := rpeu.mutation.Modified(); ok {
		_spec.SetField(rolepaneledit.FieldModified, field.TypeBool, value)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, rpeu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{rolepaneledit.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	rpeu.mutation.done = true
	return n, nil
}

// RolePanelEditUpdateOne is the builder for updating a single RolePanelEdit entity.
type RolePanelEditUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *RolePanelEditMutation
}

// SetChannelID sets the "channel_id" field.
func (rpeuo *RolePanelEditUpdateOne) SetChannelID(s snowflake.ID) *RolePanelEditUpdateOne {
	rpeuo.mutation.ResetChannelID()
	rpeuo.mutation.SetChannelID(s)
	return rpeuo
}

// SetNillableChannelID sets the "channel_id" field if the given value is not nil.
func (rpeuo *RolePanelEditUpdateOne) SetNillableChannelID(s *snowflake.ID) *RolePanelEditUpdateOne {
	if s != nil {
		rpeuo.SetChannelID(*s)
	}
	return rpeuo
}

// AddChannelID adds s to the "channel_id" field.
func (rpeuo *RolePanelEditUpdateOne) AddChannelID(s snowflake.ID) *RolePanelEditUpdateOne {
	rpeuo.mutation.AddChannelID(s)
	return rpeuo
}

// SetEmojiAuthor sets the "emoji_author" field.
func (rpeuo *RolePanelEditUpdateOne) SetEmojiAuthor(s snowflake.ID) *RolePanelEditUpdateOne {
	rpeuo.mutation.ResetEmojiAuthor()
	rpeuo.mutation.SetEmojiAuthor(s)
	return rpeuo
}

// SetNillableEmojiAuthor sets the "emoji_author" field if the given value is not nil.
func (rpeuo *RolePanelEditUpdateOne) SetNillableEmojiAuthor(s *snowflake.ID) *RolePanelEditUpdateOne {
	if s != nil {
		rpeuo.SetEmojiAuthor(*s)
	}
	return rpeuo
}

// AddEmojiAuthor adds s to the "emoji_author" field.
func (rpeuo *RolePanelEditUpdateOne) AddEmojiAuthor(s snowflake.ID) *RolePanelEditUpdateOne {
	rpeuo.mutation.AddEmojiAuthor(s)
	return rpeuo
}

// ClearEmojiAuthor clears the value of the "emoji_author" field.
func (rpeuo *RolePanelEditUpdateOne) ClearEmojiAuthor() *RolePanelEditUpdateOne {
	rpeuo.mutation.ClearEmojiAuthor()
	return rpeuo
}

// SetToken sets the "token" field.
func (rpeuo *RolePanelEditUpdateOne) SetToken(s string) *RolePanelEditUpdateOne {
	rpeuo.mutation.SetToken(s)
	return rpeuo
}

// SetNillableToken sets the "token" field if the given value is not nil.
func (rpeuo *RolePanelEditUpdateOne) SetNillableToken(s *string) *RolePanelEditUpdateOne {
	if s != nil {
		rpeuo.SetToken(*s)
	}
	return rpeuo
}

// ClearToken clears the value of the "token" field.
func (rpeuo *RolePanelEditUpdateOne) ClearToken() *RolePanelEditUpdateOne {
	rpeuo.mutation.ClearToken()
	return rpeuo
}

// SetSelectedRole sets the "selected_role" field.
func (rpeuo *RolePanelEditUpdateOne) SetSelectedRole(s snowflake.ID) *RolePanelEditUpdateOne {
	rpeuo.mutation.ResetSelectedRole()
	rpeuo.mutation.SetSelectedRole(s)
	return rpeuo
}

// SetNillableSelectedRole sets the "selected_role" field if the given value is not nil.
func (rpeuo *RolePanelEditUpdateOne) SetNillableSelectedRole(s *snowflake.ID) *RolePanelEditUpdateOne {
	if s != nil {
		rpeuo.SetSelectedRole(*s)
	}
	return rpeuo
}

// AddSelectedRole adds s to the "selected_role" field.
func (rpeuo *RolePanelEditUpdateOne) AddSelectedRole(s snowflake.ID) *RolePanelEditUpdateOne {
	rpeuo.mutation.AddSelectedRole(s)
	return rpeuo
}

// ClearSelectedRole clears the value of the "selected_role" field.
func (rpeuo *RolePanelEditUpdateOne) ClearSelectedRole() *RolePanelEditUpdateOne {
	rpeuo.mutation.ClearSelectedRole()
	return rpeuo
}

// SetModified sets the "modified" field.
func (rpeuo *RolePanelEditUpdateOne) SetModified(b bool) *RolePanelEditUpdateOne {
	rpeuo.mutation.SetModified(b)
	return rpeuo
}

// SetNillableModified sets the "modified" field if the given value is not nil.
func (rpeuo *RolePanelEditUpdateOne) SetNillableModified(b *bool) *RolePanelEditUpdateOne {
	if b != nil {
		rpeuo.SetModified(*b)
	}
	return rpeuo
}

// Mutation returns the RolePanelEditMutation object of the builder.
func (rpeuo *RolePanelEditUpdateOne) Mutation() *RolePanelEditMutation {
	return rpeuo.mutation
}

// Where appends a list predicates to the RolePanelEditUpdate builder.
func (rpeuo *RolePanelEditUpdateOne) Where(ps ...predicate.RolePanelEdit) *RolePanelEditUpdateOne {
	rpeuo.mutation.Where(ps...)
	return rpeuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (rpeuo *RolePanelEditUpdateOne) Select(field string, fields ...string) *RolePanelEditUpdateOne {
	rpeuo.fields = append([]string{field}, fields...)
	return rpeuo
}

// Save executes the query and returns the updated RolePanelEdit entity.
func (rpeuo *RolePanelEditUpdateOne) Save(ctx context.Context) (*RolePanelEdit, error) {
	return withHooks(ctx, rpeuo.sqlSave, rpeuo.mutation, rpeuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (rpeuo *RolePanelEditUpdateOne) SaveX(ctx context.Context) *RolePanelEdit {
	node, err := rpeuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (rpeuo *RolePanelEditUpdateOne) Exec(ctx context.Context) error {
	_, err := rpeuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rpeuo *RolePanelEditUpdateOne) ExecX(ctx context.Context) {
	if err := rpeuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rpeuo *RolePanelEditUpdateOne) check() error {
	if _, ok := rpeuo.mutation.GuildID(); rpeuo.mutation.GuildCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "RolePanelEdit.guild"`)
	}
	if _, ok := rpeuo.mutation.ParentID(); rpeuo.mutation.ParentCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "RolePanelEdit.parent"`)
	}
	return nil
}

func (rpeuo *RolePanelEditUpdateOne) sqlSave(ctx context.Context) (_node *RolePanelEdit, err error) {
	if err := rpeuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(rolepaneledit.Table, rolepaneledit.Columns, sqlgraph.NewFieldSpec(rolepaneledit.FieldID, field.TypeUUID))
	id, ok := rpeuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "RolePanelEdit.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := rpeuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, rolepaneledit.FieldID)
		for _, f := range fields {
			if !rolepaneledit.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != rolepaneledit.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := rpeuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := rpeuo.mutation.ChannelID(); ok {
		_spec.SetField(rolepaneledit.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := rpeuo.mutation.AddedChannelID(); ok {
		_spec.AddField(rolepaneledit.FieldChannelID, field.TypeUint64, value)
	}
	if value, ok := rpeuo.mutation.EmojiAuthor(); ok {
		_spec.SetField(rolepaneledit.FieldEmojiAuthor, field.TypeUint64, value)
	}
	if value, ok := rpeuo.mutation.AddedEmojiAuthor(); ok {
		_spec.AddField(rolepaneledit.FieldEmojiAuthor, field.TypeUint64, value)
	}
	if rpeuo.mutation.EmojiAuthorCleared() {
		_spec.ClearField(rolepaneledit.FieldEmojiAuthor, field.TypeUint64)
	}
	if value, ok := rpeuo.mutation.Token(); ok {
		_spec.SetField(rolepaneledit.FieldToken, field.TypeString, value)
	}
	if rpeuo.mutation.TokenCleared() {
		_spec.ClearField(rolepaneledit.FieldToken, field.TypeString)
	}
	if value, ok := rpeuo.mutation.SelectedRole(); ok {
		_spec.SetField(rolepaneledit.FieldSelectedRole, field.TypeUint64, value)
	}
	if value, ok := rpeuo.mutation.AddedSelectedRole(); ok {
		_spec.AddField(rolepaneledit.FieldSelectedRole, field.TypeUint64, value)
	}
	if rpeuo.mutation.SelectedRoleCleared() {
		_spec.ClearField(rolepaneledit.FieldSelectedRole, field.TypeUint64)
	}
	if value, ok := rpeuo.mutation.Modified(); ok {
		_spec.SetField(rolepaneledit.FieldModified, field.TypeBool, value)
	}
	_node = &RolePanelEdit{config: rpeuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, rpeuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{rolepaneledit.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	rpeuo.mutation.done = true
	return _node, nil
}