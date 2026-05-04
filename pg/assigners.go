package pg

import (
	"strings"
)

type PgAssigners []PgAssigner

func (a PgAssigners) SQL(queryParams *[]any) string {
	if len(a) == 0 {
		return ""
	} else if len(a) == 1 {
		return a[0].SQL(queryParams)
	}

	var sqlSt strings.Builder
	for i, assigner := range a {
		if i > 0 {
			sqlSt.WriteString(", ")
		}
		sqlSt.WriteString(assigner.SQL(queryParams))
	}

	return sqlSt.String()
}

func (a *PgAssigners) Add(fieldID string, value any) {
	*a = append(*a, PgAssigner{fieldID: fieldID, value: value})
}

func (a PgAssigners) Len() int {
	return len(a)
}
