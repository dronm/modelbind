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

// Set replaces a column assignment while preserving its position and removing
// duplicate entries. Only unqualified write columns are accepted.
func (a *PgAssigners) Set(fieldID string, value any) error {
	column, err := writeColumn(fieldID)
	if err != nil {
		return err
	}
	result := make(PgAssigners, 0, len(*a)+1)
	replaced := false
	for _, assignment := range *a {
		if strings.EqualFold(strings.TrimSpace(assignment.fieldID), column) {
			if !replaced {
				result = append(result, PgAssigner{fieldID: column, value: value})
				replaced = true
			}
			continue
		}
		result = append(result, assignment)
	}
	if !replaced {
		result = append(result, PgAssigner{fieldID: column, value: value})
	}
	*a = result
	return nil
}
