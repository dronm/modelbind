package pg

import (
	"fmt"
	"strings"

	"github.com/dronm/modelbind/types"
)

// FilterExpression is a closed tree of collection conditions. Its zero value is
// invalid. Leaves retain PgFilter semantics, including array and FTS operators.
// It is separate from access.Predicate because collection NULL and operator
// semantics must not change the authorization predicate contract.
type FilterExpression struct {
	leaf     *PgFilter
	join     types.FilterJoin
	children []FilterExpression
}

// FilterLeaf snapshots one trusted, server-built collection condition. Clients
// must go through modelbind.ParseCollectionWhere for metadata validation.
func FilterLeaf(filters PgFilters) FilterExpression {
	if len(filters) != 1 {
		return FilterExpression{}
	}
	leaf := filters[0]
	return FilterExpression{leaf: &leaf}
}

func FilterAnd(children ...FilterExpression) FilterExpression {
	return FilterExpression{join: types.SQLFilterJoinAnd, children: append([]FilterExpression(nil), children...)}
}

func FilterOr(children ...FilterExpression) FilterExpression {
	return FilterExpression{join: types.SQLFilterJoinOr, children: append([]FilterExpression(nil), children...)}
}

func (e FilterExpression) validate(depth int) error {
	if depth > 64 {
		return fmt.Errorf("where: nesting exceeds 64 levels")
	}
	if e.leaf != nil {
		return nil
	}
	if len(e.children) == 0 || (e.join != types.SQLFilterJoinAnd && e.join != types.SQLFilterJoinOr) {
		return fmt.Errorf("where: missing condition or empty group")
	}
	for _, child := range e.children {
		if err := child.validate(depth + 1); err != nil {
			return err
		}
	}
	return nil
}

func (e FilterExpression) sql(params *[]any) string {
	if e.leaf != nil {
		return e.leaf.SQL(params)
	}
	parts := make([]string, len(e.children))
	for i, child := range e.children {
		parts[i] = "(" + child.sql(params) + ")"
	}
	return strings.Join(parts, " "+string(e.join)+" ")
}

// PgGroupedSelect is the opt-in collection builder. Embedding keeps policies,
// model binding, sorters and limits on the existing PgSelect. No legacy struct
// or interface is extended. Use this wrapper's SQL methods to include Where.
type PgGroupedSelect struct {
	*PgSelect
	where *FilterExpression
	err   error
}

func NewPgGroupedSelect(model types.DBAggModel, filter *PgFilters, sorter *PgSorters, limit *PgLimit) *PgGroupedSelect {
	return &PgGroupedSelect{PgSelect: NewPgSelect(model, filter, sorter, limit)}
}

// SetWhere never replaces a server policy or request predicate. A failed set is
// remembered so ignoring its error cannot generate an unrestricted query.
func (s *PgGroupedSelect) SetWhere(expression FilterExpression) error {
	s.where = &expression
	s.err = expression.validate(0)
	return s.err
}

func (s PgGroupedSelect) SQL(params *[]any) string {
	return mustSQL(s.BuildSQL(params))
}

func (s PgGroupedSelect) BuildSQL(params *[]any) (string, error) {
	query, _, err := s.build(params, false)
	return query, err
}

func (s PgGroupedSelect) CollectionSQL(params *[]any) (string, string) {
	query, aggregate, err := s.BuildCollectionSQL(params)
	if err != nil {
		panic(err)
	}
	return query, aggregate
}

func (s PgGroupedSelect) BuildCollectionSQL(params *[]any) (string, string, error) {
	return s.build(params, true)
}

func (s PgGroupedSelect) build(params *[]any, totals bool) (string, string, error) {
	if s.err != nil {
		return "", "", s.err
	}
	if s.PgSelect == nil {
		return "", "", fmt.Errorf("grouped select is not initialized")
	}
	// Exact legacy execution path when the new expression is absent.
	if s.where == nil {
		if totals {
			return s.PgSelect.BuildCollectionSQL(params)
		}
		query, err := s.PgSelect.BuildSQL(params)
		return query, "", err
	}
	aggregate := ""
	query, err := buildStatement(params, func(args *[]any) (string, error) {
		// Keep the complete flat legacy expression in its original precedence.
		// scopedWhere also keeps caller predicates and policies independent.
		base, err := scopedWhere(s.filter, s.request, s.policy, args)
		if err != nil {
			return "", err
		}
		grouped := s.where.sql(args)
		where := " WHERE (" + grouped + ")"
		if base != "" {
			where = " WHERE (" + strings.TrimPrefix(base, " WHERE ") + ") AND (" + grouped + ")"
		}
		if totals && len(s.aggFields) > 0 {
			aggregate = fmt.Sprintf("SELECT %s FROM %s%s", strings.Join(s.aggFields, ","), s.model.Relation(), where)
		}
		return s.selectSQL(where), nil
	})
	if err != nil {
		return "", "", err
	}
	return query, aggregate, nil
}
