package pg

import (
	"fmt"
	"strings"

	"github.com/dronm/modelbind/types"
)

type PgSelect struct {
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

func (s PgSelect) SQL(queryParams *[]any) string {
	var filterSQL string
	if s.filter != nil {
		filterSQL = s.filter.SQL(queryParams)
	}
	var sorterSQL string
	if s.sorter != nil {
		sorterSQL = s.sorter.SQL()
	}
	var limitSQL string
	if s.limit != nil {
		limitSQL = s.limit.SQL()
	}
	return fmt.Sprintf("SELECT %s FROM %s%s%s%s",
		strings.Join(s.fieldIds, ","),
		s.model.Relation(),
		filterSQL,
		sorterSQL,
		limitSQL,
	)
}

// CollectionSQL returns two queries: collecion query and aggregation query.
func (s PgSelect) CollectionSQL(queryParams *[]any) (string, string) {
	var filterSQL string
	if s.filter != nil {
		filterSQL = s.filter.SQL(queryParams)
	}
	var sorterSQL string
	if s.sorter != nil {
		sorterSQL = s.sorter.SQL()
	}
	var limitSQL string
	if s.limit != nil {
		limitSQL = s.limit.SQL()
	}

	totQuery := ""
	if len(s.aggFields) > 0 {
		totQuery = fmt.Sprintf("SELECT %s FROM %s%s",
			strings.Join(s.aggFields, ","),
			s.model.Relation(),
			filterSQL,
		)
	}

	return fmt.Sprintf("SELECT %s FROM %s%s%s%s",
			strings.Join(s.fieldIds, ","),
			s.model.Relation(),
			filterSQL,
			sorterSQL,
			limitSQL,
		),
		totQuery
}
