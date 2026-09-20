package pg

import (
	"fmt"
	"strings"

	"github.com/dronm/modelbind/types"
)

type PgSelect struct {
	policy         *boundPolicy
	request        *boundPredicate
	model          types.DBAggModel
	filter         *PgFilters
	sorter         *PgSorters
	limit          *PgLimit
	fieldIds       []string
	fieldValues    []any
	aggFields      []string
	aggFieldValues []any
}

func NewPgSelect(model types.DBAggModel, filter *PgFilters, sorter *PgSorters, limit *PgLimit) *PgSelect {
	return &PgSelect{model: model,
		filter: filter,
		sorter: sorter,
		limit:  limit,
	}
}

func (s PgSelect) Model() types.DBAggModel {
	return s.model
}

func (s PgSelect) Filter() types.DBFilters {
	return s.filter
}

func (s *PgSelect) SetFilter(f types.DBFilters) error {
	filters, ok := f.(*PgFilters)
	if !ok {
		return fmt.Errorf("could not assert to *PgFilters")
	}
	s.filter = filters
	return nil
}

func (s PgSelect) Limit() types.DBLimit {
	return s.limit
}

func (s PgSelect) Sorter() types.DBSorters {
	return s.sorter
}

func (s PgSelect) FieldValues() []any {
	return s.fieldValues
}

func (s *PgSelect) AddField(id string, val any) {
	s.fieldIds = append(s.fieldIds, id)
	s.fieldValues = append(s.fieldValues, val)
}

// AddAggField adds aggregate function, fn is the function,
// val is the value for scaning result.
func (s *PgSelect) AddAggField(fn string, val any) {
	s.aggFields = append(s.aggFields, fn)
	s.aggFieldValues = append(s.aggFieldValues, val)
}

// SQL preserves the legacy API and panics on build errors. Prefer BuildSQL for
// policy-aware code so authorization/configuration failures can be handled.
func (s PgSelect) SQL(queryParams *[]any) string {
	return mustSQL(s.BuildSQL(queryParams))
}

func (s PgSelect) BuildSQL(queryParams *[]any) (string, error) {
	return buildStatement(queryParams, func(params *[]any) (string, error) {
		filterSQL, err := scopedWhere(s.filter, s.request, s.policy, params)
		if err != nil {
			return "", err
		}
		return s.selectSQL(filterSQL), nil
	})
}

func (s PgSelect) selectSQL(filterSQL string) string {
	var sorterSQL, limitSQL string
	if s.sorter != nil {
		sorterSQL = s.sorter.SQL()
	}
	if s.limit != nil {
		limitSQL = s.limit.SQL()
	}
	return fmt.Sprintf("SELECT %s FROM %s%s%s%s",
		strings.Join(s.fieldIds, ","), s.model.Relation(), filterSQL, sorterSQL, limitSQL)
}

// CollectionSQL returns list and aggregate queries sharing the same parameters.
func (s PgSelect) CollectionSQL(queryParams *[]any) (string, string) {
	query, aggregate, err := s.BuildCollectionSQL(queryParams)
	if err != nil {
		panic(err)
	}
	return query, aggregate
}

// BuildCollectionSQL applies the identical row scope to data and totals. Policy
// parameters are allocated once, not separately for the aggregate query.
func (s PgSelect) BuildCollectionSQL(queryParams *[]any) (string, string, error) {
	aggregate := ""
	query, err := buildStatement(queryParams, func(params *[]any) (string, error) {
		filterSQL, err := scopedWhere(s.filter, s.request, s.policy, params)
		if err != nil {
			return "", err
		}
		if len(s.aggFields) > 0 {
			aggregate = fmt.Sprintf("SELECT %s FROM %s%s", strings.Join(s.aggFields, ","), s.model.Relation(), filterSQL)
		}
		return s.selectSQL(filterSQL), nil
	})
	if err != nil {
		return "", "", err
	}
	return query, aggregate, nil
}
