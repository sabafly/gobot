// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/messageremind"
	"github.com/sabafly/gobot/ent/predicate"
)

// MessageRemindQuery is the builder for querying MessageRemind entities.
type MessageRemindQuery struct {
	config
	ctx        *QueryContext
	order      []messageremind.OrderOption
	inters     []Interceptor
	predicates []predicate.MessageRemind
	withGuild  *GuildQuery
	withFKs    bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the MessageRemindQuery builder.
func (mrq *MessageRemindQuery) Where(ps ...predicate.MessageRemind) *MessageRemindQuery {
	mrq.predicates = append(mrq.predicates, ps...)
	return mrq
}

// Limit the number of records to be returned by this query.
func (mrq *MessageRemindQuery) Limit(limit int) *MessageRemindQuery {
	mrq.ctx.Limit = &limit
	return mrq
}

// Offset to start from.
func (mrq *MessageRemindQuery) Offset(offset int) *MessageRemindQuery {
	mrq.ctx.Offset = &offset
	return mrq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (mrq *MessageRemindQuery) Unique(unique bool) *MessageRemindQuery {
	mrq.ctx.Unique = &unique
	return mrq
}

// Order specifies how the records should be ordered.
func (mrq *MessageRemindQuery) Order(o ...messageremind.OrderOption) *MessageRemindQuery {
	mrq.order = append(mrq.order, o...)
	return mrq
}

// QueryGuild chains the current query on the "guild" edge.
func (mrq *MessageRemindQuery) QueryGuild() *GuildQuery {
	query := (&GuildClient{config: mrq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := mrq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := mrq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(messageremind.Table, messageremind.FieldID, selector),
			sqlgraph.To(guild.Table, guild.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, messageremind.GuildTable, messageremind.GuildColumn),
		)
		fromU = sqlgraph.SetNeighbors(mrq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first MessageRemind entity from the query.
// Returns a *NotFoundError when no MessageRemind was found.
func (mrq *MessageRemindQuery) First(ctx context.Context) (*MessageRemind, error) {
	nodes, err := mrq.Limit(1).All(setContextOp(ctx, mrq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{messageremind.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (mrq *MessageRemindQuery) FirstX(ctx context.Context) *MessageRemind {
	node, err := mrq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first MessageRemind ID from the query.
// Returns a *NotFoundError when no MessageRemind ID was found.
func (mrq *MessageRemindQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = mrq.Limit(1).IDs(setContextOp(ctx, mrq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{messageremind.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (mrq *MessageRemindQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := mrq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single MessageRemind entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one MessageRemind entity is found.
// Returns a *NotFoundError when no MessageRemind entities are found.
func (mrq *MessageRemindQuery) Only(ctx context.Context) (*MessageRemind, error) {
	nodes, err := mrq.Limit(2).All(setContextOp(ctx, mrq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{messageremind.Label}
	default:
		return nil, &NotSingularError{messageremind.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (mrq *MessageRemindQuery) OnlyX(ctx context.Context) *MessageRemind {
	node, err := mrq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only MessageRemind ID in the query.
// Returns a *NotSingularError when more than one MessageRemind ID is found.
// Returns a *NotFoundError when no entities are found.
func (mrq *MessageRemindQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = mrq.Limit(2).IDs(setContextOp(ctx, mrq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{messageremind.Label}
	default:
		err = &NotSingularError{messageremind.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (mrq *MessageRemindQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := mrq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of MessageReminds.
func (mrq *MessageRemindQuery) All(ctx context.Context) ([]*MessageRemind, error) {
	ctx = setContextOp(ctx, mrq.ctx, "All")
	if err := mrq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*MessageRemind, *MessageRemindQuery]()
	return withInterceptors[[]*MessageRemind](ctx, mrq, qr, mrq.inters)
}

// AllX is like All, but panics if an error occurs.
func (mrq *MessageRemindQuery) AllX(ctx context.Context) []*MessageRemind {
	nodes, err := mrq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of MessageRemind IDs.
func (mrq *MessageRemindQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if mrq.ctx.Unique == nil && mrq.path != nil {
		mrq.Unique(true)
	}
	ctx = setContextOp(ctx, mrq.ctx, "IDs")
	if err = mrq.Select(messageremind.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (mrq *MessageRemindQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := mrq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (mrq *MessageRemindQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, mrq.ctx, "Count")
	if err := mrq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, mrq, querierCount[*MessageRemindQuery](), mrq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (mrq *MessageRemindQuery) CountX(ctx context.Context) int {
	count, err := mrq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (mrq *MessageRemindQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, mrq.ctx, "Exist")
	switch _, err := mrq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (mrq *MessageRemindQuery) ExistX(ctx context.Context) bool {
	exist, err := mrq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the MessageRemindQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (mrq *MessageRemindQuery) Clone() *MessageRemindQuery {
	if mrq == nil {
		return nil
	}
	return &MessageRemindQuery{
		config:     mrq.config,
		ctx:        mrq.ctx.Clone(),
		order:      append([]messageremind.OrderOption{}, mrq.order...),
		inters:     append([]Interceptor{}, mrq.inters...),
		predicates: append([]predicate.MessageRemind{}, mrq.predicates...),
		withGuild:  mrq.withGuild.Clone(),
		// clone intermediate query.
		sql:  mrq.sql.Clone(),
		path: mrq.path,
	}
}

// WithGuild tells the query-builder to eager-load the nodes that are connected to
// the "guild" edge. The optional arguments are used to configure the query builder of the edge.
func (mrq *MessageRemindQuery) WithGuild(opts ...func(*GuildQuery)) *MessageRemindQuery {
	query := (&GuildClient{config: mrq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	mrq.withGuild = query
	return mrq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		ChannelID snowflake.ID `json:"channel_id,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.MessageRemind.Query().
//		GroupBy(messageremind.FieldChannelID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (mrq *MessageRemindQuery) GroupBy(field string, fields ...string) *MessageRemindGroupBy {
	mrq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &MessageRemindGroupBy{build: mrq}
	grbuild.flds = &mrq.ctx.Fields
	grbuild.label = messageremind.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		ChannelID snowflake.ID `json:"channel_id,omitempty"`
//	}
//
//	client.MessageRemind.Query().
//		Select(messageremind.FieldChannelID).
//		Scan(ctx, &v)
func (mrq *MessageRemindQuery) Select(fields ...string) *MessageRemindSelect {
	mrq.ctx.Fields = append(mrq.ctx.Fields, fields...)
	sbuild := &MessageRemindSelect{MessageRemindQuery: mrq}
	sbuild.label = messageremind.Label
	sbuild.flds, sbuild.scan = &mrq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a MessageRemindSelect configured with the given aggregations.
func (mrq *MessageRemindQuery) Aggregate(fns ...AggregateFunc) *MessageRemindSelect {
	return mrq.Select().Aggregate(fns...)
}

func (mrq *MessageRemindQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range mrq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, mrq); err != nil {
				return err
			}
		}
	}
	for _, f := range mrq.ctx.Fields {
		if !messageremind.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if mrq.path != nil {
		prev, err := mrq.path(ctx)
		if err != nil {
			return err
		}
		mrq.sql = prev
	}
	return nil
}

func (mrq *MessageRemindQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*MessageRemind, error) {
	var (
		nodes       = []*MessageRemind{}
		withFKs     = mrq.withFKs
		_spec       = mrq.querySpec()
		loadedTypes = [1]bool{
			mrq.withGuild != nil,
		}
	)
	if mrq.withGuild != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, messageremind.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*MessageRemind).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &MessageRemind{config: mrq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, mrq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := mrq.withGuild; query != nil {
		if err := mrq.loadGuild(ctx, query, nodes, nil,
			func(n *MessageRemind, e *Guild) { n.Edges.Guild = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (mrq *MessageRemindQuery) loadGuild(ctx context.Context, query *GuildQuery, nodes []*MessageRemind, init func(*MessageRemind), assign func(*MessageRemind, *Guild)) error {
	ids := make([]snowflake.ID, 0, len(nodes))
	nodeids := make(map[snowflake.ID][]*MessageRemind)
	for i := range nodes {
		if nodes[i].guild_reminds == nil {
			continue
		}
		fk := *nodes[i].guild_reminds
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(guild.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "guild_reminds" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (mrq *MessageRemindQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := mrq.querySpec()
	_spec.Node.Columns = mrq.ctx.Fields
	if len(mrq.ctx.Fields) > 0 {
		_spec.Unique = mrq.ctx.Unique != nil && *mrq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, mrq.driver, _spec)
}

func (mrq *MessageRemindQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(messageremind.Table, messageremind.Columns, sqlgraph.NewFieldSpec(messageremind.FieldID, field.TypeUUID))
	_spec.From = mrq.sql
	if unique := mrq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if mrq.path != nil {
		_spec.Unique = true
	}
	if fields := mrq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, messageremind.FieldID)
		for i := range fields {
			if fields[i] != messageremind.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := mrq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := mrq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := mrq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := mrq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (mrq *MessageRemindQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(mrq.driver.Dialect())
	t1 := builder.Table(messageremind.Table)
	columns := mrq.ctx.Fields
	if len(columns) == 0 {
		columns = messageremind.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if mrq.sql != nil {
		selector = mrq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if mrq.ctx.Unique != nil && *mrq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range mrq.predicates {
		p(selector)
	}
	for _, p := range mrq.order {
		p(selector)
	}
	if offset := mrq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := mrq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// MessageRemindGroupBy is the group-by builder for MessageRemind entities.
type MessageRemindGroupBy struct {
	selector
	build *MessageRemindQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (mrgb *MessageRemindGroupBy) Aggregate(fns ...AggregateFunc) *MessageRemindGroupBy {
	mrgb.fns = append(mrgb.fns, fns...)
	return mrgb
}

// Scan applies the selector query and scans the result into the given value.
func (mrgb *MessageRemindGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, mrgb.build.ctx, "GroupBy")
	if err := mrgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*MessageRemindQuery, *MessageRemindGroupBy](ctx, mrgb.build, mrgb, mrgb.build.inters, v)
}

func (mrgb *MessageRemindGroupBy) sqlScan(ctx context.Context, root *MessageRemindQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(mrgb.fns))
	for _, fn := range mrgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*mrgb.flds)+len(mrgb.fns))
		for _, f := range *mrgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*mrgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := mrgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// MessageRemindSelect is the builder for selecting fields of MessageRemind entities.
type MessageRemindSelect struct {
	*MessageRemindQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (mrs *MessageRemindSelect) Aggregate(fns ...AggregateFunc) *MessageRemindSelect {
	mrs.fns = append(mrs.fns, fns...)
	return mrs
}

// Scan applies the selector query and scans the result into the given value.
func (mrs *MessageRemindSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, mrs.ctx, "Select")
	if err := mrs.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*MessageRemindQuery, *MessageRemindSelect](ctx, mrs.MessageRemindQuery, mrs, mrs.inters, v)
}

func (mrs *MessageRemindSelect) sqlScan(ctx context.Context, root *MessageRemindQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(mrs.fns))
	for _, fn := range mrs.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*mrs.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := mrs.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
