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
	"github.com/sabafly/gobot/ent/chinchiroplayer"
	"github.com/sabafly/gobot/ent/chinchirosession"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/user"
)

// ChinchiroPlayerQuery is the builder for querying ChinchiroPlayer entities.
type ChinchiroPlayerQuery struct {
	config
	ctx         *QueryContext
	order       []chinchiroplayer.OrderOption
	inters      []Interceptor
	predicates  []predicate.ChinchiroPlayer
	withUser    *UserQuery
	withSession *ChinchiroSessionQuery
	withFKs     bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the ChinchiroPlayerQuery builder.
func (cpq *ChinchiroPlayerQuery) Where(ps ...predicate.ChinchiroPlayer) *ChinchiroPlayerQuery {
	cpq.predicates = append(cpq.predicates, ps...)
	return cpq
}

// Limit the number of records to be returned by this query.
func (cpq *ChinchiroPlayerQuery) Limit(limit int) *ChinchiroPlayerQuery {
	cpq.ctx.Limit = &limit
	return cpq
}

// Offset to start from.
func (cpq *ChinchiroPlayerQuery) Offset(offset int) *ChinchiroPlayerQuery {
	cpq.ctx.Offset = &offset
	return cpq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (cpq *ChinchiroPlayerQuery) Unique(unique bool) *ChinchiroPlayerQuery {
	cpq.ctx.Unique = &unique
	return cpq
}

// Order specifies how the records should be ordered.
func (cpq *ChinchiroPlayerQuery) Order(o ...chinchiroplayer.OrderOption) *ChinchiroPlayerQuery {
	cpq.order = append(cpq.order, o...)
	return cpq
}

// QueryUser chains the current query on the "user" edge.
func (cpq *ChinchiroPlayerQuery) QueryUser() *UserQuery {
	query := (&UserClient{config: cpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(chinchiroplayer.Table, chinchiroplayer.FieldID, selector),
			sqlgraph.To(user.Table, user.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, chinchiroplayer.UserTable, chinchiroplayer.UserColumn),
		)
		fromU = sqlgraph.SetNeighbors(cpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QuerySession chains the current query on the "session" edge.
func (cpq *ChinchiroPlayerQuery) QuerySession() *ChinchiroSessionQuery {
	query := (&ChinchiroSessionClient{config: cpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(chinchiroplayer.Table, chinchiroplayer.FieldID, selector),
			sqlgraph.To(chinchirosession.Table, chinchirosession.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, chinchiroplayer.SessionTable, chinchiroplayer.SessionColumn),
		)
		fromU = sqlgraph.SetNeighbors(cpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first ChinchiroPlayer entity from the query.
// Returns a *NotFoundError when no ChinchiroPlayer was found.
func (cpq *ChinchiroPlayerQuery) First(ctx context.Context) (*ChinchiroPlayer, error) {
	nodes, err := cpq.Limit(1).All(setContextOp(ctx, cpq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{chinchiroplayer.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (cpq *ChinchiroPlayerQuery) FirstX(ctx context.Context) *ChinchiroPlayer {
	node, err := cpq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first ChinchiroPlayer ID from the query.
// Returns a *NotFoundError when no ChinchiroPlayer ID was found.
func (cpq *ChinchiroPlayerQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = cpq.Limit(1).IDs(setContextOp(ctx, cpq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{chinchiroplayer.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (cpq *ChinchiroPlayerQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := cpq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single ChinchiroPlayer entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one ChinchiroPlayer entity is found.
// Returns a *NotFoundError when no ChinchiroPlayer entities are found.
func (cpq *ChinchiroPlayerQuery) Only(ctx context.Context) (*ChinchiroPlayer, error) {
	nodes, err := cpq.Limit(2).All(setContextOp(ctx, cpq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{chinchiroplayer.Label}
	default:
		return nil, &NotSingularError{chinchiroplayer.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (cpq *ChinchiroPlayerQuery) OnlyX(ctx context.Context) *ChinchiroPlayer {
	node, err := cpq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only ChinchiroPlayer ID in the query.
// Returns a *NotSingularError when more than one ChinchiroPlayer ID is found.
// Returns a *NotFoundError when no entities are found.
func (cpq *ChinchiroPlayerQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = cpq.Limit(2).IDs(setContextOp(ctx, cpq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{chinchiroplayer.Label}
	default:
		err = &NotSingularError{chinchiroplayer.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (cpq *ChinchiroPlayerQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := cpq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of ChinchiroPlayers.
func (cpq *ChinchiroPlayerQuery) All(ctx context.Context) ([]*ChinchiroPlayer, error) {
	ctx = setContextOp(ctx, cpq.ctx, "All")
	if err := cpq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*ChinchiroPlayer, *ChinchiroPlayerQuery]()
	return withInterceptors[[]*ChinchiroPlayer](ctx, cpq, qr, cpq.inters)
}

// AllX is like All, but panics if an error occurs.
func (cpq *ChinchiroPlayerQuery) AllX(ctx context.Context) []*ChinchiroPlayer {
	nodes, err := cpq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of ChinchiroPlayer IDs.
func (cpq *ChinchiroPlayerQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if cpq.ctx.Unique == nil && cpq.path != nil {
		cpq.Unique(true)
	}
	ctx = setContextOp(ctx, cpq.ctx, "IDs")
	if err = cpq.Select(chinchiroplayer.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (cpq *ChinchiroPlayerQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := cpq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (cpq *ChinchiroPlayerQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, cpq.ctx, "Count")
	if err := cpq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, cpq, querierCount[*ChinchiroPlayerQuery](), cpq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (cpq *ChinchiroPlayerQuery) CountX(ctx context.Context) int {
	count, err := cpq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (cpq *ChinchiroPlayerQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, cpq.ctx, "Exist")
	switch _, err := cpq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (cpq *ChinchiroPlayerQuery) ExistX(ctx context.Context) bool {
	exist, err := cpq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the ChinchiroPlayerQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (cpq *ChinchiroPlayerQuery) Clone() *ChinchiroPlayerQuery {
	if cpq == nil {
		return nil
	}
	return &ChinchiroPlayerQuery{
		config:      cpq.config,
		ctx:         cpq.ctx.Clone(),
		order:       append([]chinchiroplayer.OrderOption{}, cpq.order...),
		inters:      append([]Interceptor{}, cpq.inters...),
		predicates:  append([]predicate.ChinchiroPlayer{}, cpq.predicates...),
		withUser:    cpq.withUser.Clone(),
		withSession: cpq.withSession.Clone(),
		// clone intermediate query.
		sql:  cpq.sql.Clone(),
		path: cpq.path,
	}
}

// WithUser tells the query-builder to eager-load the nodes that are connected to
// the "user" edge. The optional arguments are used to configure the query builder of the edge.
func (cpq *ChinchiroPlayerQuery) WithUser(opts ...func(*UserQuery)) *ChinchiroPlayerQuery {
	query := (&UserClient{config: cpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cpq.withUser = query
	return cpq
}

// WithSession tells the query-builder to eager-load the nodes that are connected to
// the "session" edge. The optional arguments are used to configure the query builder of the edge.
func (cpq *ChinchiroPlayerQuery) WithSession(opts ...func(*ChinchiroSessionQuery)) *ChinchiroPlayerQuery {
	query := (&ChinchiroSessionClient{config: cpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cpq.withSession = query
	return cpq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Point int `json:"point,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.ChinchiroPlayer.Query().
//		GroupBy(chinchiroplayer.FieldPoint).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (cpq *ChinchiroPlayerQuery) GroupBy(field string, fields ...string) *ChinchiroPlayerGroupBy {
	cpq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &ChinchiroPlayerGroupBy{build: cpq}
	grbuild.flds = &cpq.ctx.Fields
	grbuild.label = chinchiroplayer.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Point int `json:"point,omitempty"`
//	}
//
//	client.ChinchiroPlayer.Query().
//		Select(chinchiroplayer.FieldPoint).
//		Scan(ctx, &v)
func (cpq *ChinchiroPlayerQuery) Select(fields ...string) *ChinchiroPlayerSelect {
	cpq.ctx.Fields = append(cpq.ctx.Fields, fields...)
	sbuild := &ChinchiroPlayerSelect{ChinchiroPlayerQuery: cpq}
	sbuild.label = chinchiroplayer.Label
	sbuild.flds, sbuild.scan = &cpq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a ChinchiroPlayerSelect configured with the given aggregations.
func (cpq *ChinchiroPlayerQuery) Aggregate(fns ...AggregateFunc) *ChinchiroPlayerSelect {
	return cpq.Select().Aggregate(fns...)
}

func (cpq *ChinchiroPlayerQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range cpq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, cpq); err != nil {
				return err
			}
		}
	}
	for _, f := range cpq.ctx.Fields {
		if !chinchiroplayer.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if cpq.path != nil {
		prev, err := cpq.path(ctx)
		if err != nil {
			return err
		}
		cpq.sql = prev
	}
	return nil
}

func (cpq *ChinchiroPlayerQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*ChinchiroPlayer, error) {
	var (
		nodes       = []*ChinchiroPlayer{}
		withFKs     = cpq.withFKs
		_spec       = cpq.querySpec()
		loadedTypes = [2]bool{
			cpq.withUser != nil,
			cpq.withSession != nil,
		}
	)
	if cpq.withSession != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, chinchiroplayer.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*ChinchiroPlayer).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &ChinchiroPlayer{config: cpq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, cpq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := cpq.withUser; query != nil {
		if err := cpq.loadUser(ctx, query, nodes, nil,
			func(n *ChinchiroPlayer, e *User) { n.Edges.User = e }); err != nil {
			return nil, err
		}
	}
	if query := cpq.withSession; query != nil {
		if err := cpq.loadSession(ctx, query, nodes, nil,
			func(n *ChinchiroPlayer, e *ChinchiroSession) { n.Edges.Session = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (cpq *ChinchiroPlayerQuery) loadUser(ctx context.Context, query *UserQuery, nodes []*ChinchiroPlayer, init func(*ChinchiroPlayer), assign func(*ChinchiroPlayer, *User)) error {
	ids := make([]snowflake.ID, 0, len(nodes))
	nodeids := make(map[snowflake.ID][]*ChinchiroPlayer)
	for i := range nodes {
		fk := nodes[i].UserID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(user.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "user_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (cpq *ChinchiroPlayerQuery) loadSession(ctx context.Context, query *ChinchiroSessionQuery, nodes []*ChinchiroPlayer, init func(*ChinchiroPlayer), assign func(*ChinchiroPlayer, *ChinchiroSession)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*ChinchiroPlayer)
	for i := range nodes {
		if nodes[i].chinchiro_session_players == nil {
			continue
		}
		fk := *nodes[i].chinchiro_session_players
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(chinchirosession.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "chinchiro_session_players" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (cpq *ChinchiroPlayerQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := cpq.querySpec()
	_spec.Node.Columns = cpq.ctx.Fields
	if len(cpq.ctx.Fields) > 0 {
		_spec.Unique = cpq.ctx.Unique != nil && *cpq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, cpq.driver, _spec)
}

func (cpq *ChinchiroPlayerQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(chinchiroplayer.Table, chinchiroplayer.Columns, sqlgraph.NewFieldSpec(chinchiroplayer.FieldID, field.TypeUUID))
	_spec.From = cpq.sql
	if unique := cpq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if cpq.path != nil {
		_spec.Unique = true
	}
	if fields := cpq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, chinchiroplayer.FieldID)
		for i := range fields {
			if fields[i] != chinchiroplayer.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if cpq.withUser != nil {
			_spec.Node.AddColumnOnce(chinchiroplayer.FieldUserID)
		}
	}
	if ps := cpq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := cpq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := cpq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := cpq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (cpq *ChinchiroPlayerQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(cpq.driver.Dialect())
	t1 := builder.Table(chinchiroplayer.Table)
	columns := cpq.ctx.Fields
	if len(columns) == 0 {
		columns = chinchiroplayer.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if cpq.sql != nil {
		selector = cpq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if cpq.ctx.Unique != nil && *cpq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range cpq.predicates {
		p(selector)
	}
	for _, p := range cpq.order {
		p(selector)
	}
	if offset := cpq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := cpq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ChinchiroPlayerGroupBy is the group-by builder for ChinchiroPlayer entities.
type ChinchiroPlayerGroupBy struct {
	selector
	build *ChinchiroPlayerQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (cpgb *ChinchiroPlayerGroupBy) Aggregate(fns ...AggregateFunc) *ChinchiroPlayerGroupBy {
	cpgb.fns = append(cpgb.fns, fns...)
	return cpgb
}

// Scan applies the selector query and scans the result into the given value.
func (cpgb *ChinchiroPlayerGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, cpgb.build.ctx, "GroupBy")
	if err := cpgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*ChinchiroPlayerQuery, *ChinchiroPlayerGroupBy](ctx, cpgb.build, cpgb, cpgb.build.inters, v)
}

func (cpgb *ChinchiroPlayerGroupBy) sqlScan(ctx context.Context, root *ChinchiroPlayerQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(cpgb.fns))
	for _, fn := range cpgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*cpgb.flds)+len(cpgb.fns))
		for _, f := range *cpgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*cpgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := cpgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// ChinchiroPlayerSelect is the builder for selecting fields of ChinchiroPlayer entities.
type ChinchiroPlayerSelect struct {
	*ChinchiroPlayerQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (cps *ChinchiroPlayerSelect) Aggregate(fns ...AggregateFunc) *ChinchiroPlayerSelect {
	cps.fns = append(cps.fns, fns...)
	return cps
}

// Scan applies the selector query and scans the result into the given value.
func (cps *ChinchiroPlayerSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, cps.ctx, "Select")
	if err := cps.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*ChinchiroPlayerQuery, *ChinchiroPlayerSelect](ctx, cps.ChinchiroPlayerQuery, cps, cps.inters, v)
}

func (cps *ChinchiroPlayerSelect) sqlScan(ctx context.Context, root *ChinchiroPlayerQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(cps.fns))
	for _, fn := range cps.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*cps.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := cps.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
