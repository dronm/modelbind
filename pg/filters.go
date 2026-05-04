package pg

import (
	"fmt"
	"strings"

	"github.com/dronm/modelbind/types"
)

type PgFilters []PgFilter

func (f *PgFilters) Add(pref, fieldID string, value any, operator types.SQLFilterOperator, join types.FilterJoin) {
	*f = append(*f, PgFilter{fieldID: fieldID, value: value, join: join, operator: operator, fieldPref: pref})
}

func (f *PgFilters) AddFullTextSearch(pref, fieldID string, value any, join types.FilterJoin) {
	safeFieldID, err := sanitizeSQLFieldRef(fieldID)
	if err != nil {
		panic(err)
	}
	if pref != "" {
		safeFieldID = pref + "." + safeFieldID
	}
	*f = append(*f, PgFilter{
		value:      value,
		join:       join,
		expression: fmt.Sprintf("%s @@ to_tsquery('russian', {{PARAM}})", safeFieldID),
	})
}

func (f *PgFilters) AddArrayInclude(pref, fieldID string, value any, join types.FilterJoin) {
	safeFieldID, err := sanitizeSQLFieldRef(fieldID)
	if err != nil {
		panic(err)
	}
	if pref != "" {
		safeFieldID = pref + "." + safeFieldID
	}
	*f = append(*f, PgFilter{
		value:      value,
		join:       join,
		expression: fmt.Sprintf("%s = ANY({{PARAM}})", safeFieldID),
	})
}

func (f *PgFilters) AddColumnArrayInclude(pref, fieldID string, value any, join types.FilterJoin) {
	safeFieldID, err := sanitizeSQLFieldRef(fieldID)
	if err != nil {
		panic(err)
	}
	if pref != "" {
		safeFieldID = pref + "." + safeFieldID
	}
	*f = append(*f, PgFilter{
		value:      value,
		join:       join,
		expression: fmt.Sprintf("{{PARAM}} = ANY(%s)", safeFieldID),
	})
}

func (f PgFilters) Len() int {
	return len(f)
}

func (f PgFilters) SQL(queryParams *[]any) string {
	if len(f) == 0 {
		return ""
	} else if len(f) == 1 {
		return " WHERE " + f[0].SQL(queryParams)
	}

	var sqlSt strings.Builder
	sqlSt.WriteString(" WHERE ")
	for i, filter := range f {
		if i > 0 {
			if filter.join == "" {
				filter.join = types.SQLFilterJoinAnd
			}
			sqlSt.WriteString(" " + string(filter.join) + " ")
		}
		sqlSt.WriteString("(" + filter.SQL(queryParams) + ")")
	}
	return sqlSt.String()
}
