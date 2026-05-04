package pg

import (
	"strings"

	"github.com/dronm/modelbind/types"
)

type PgSorters []PgSorter

func (s *PgSorters) Add(fieldId string, direct types.SQLSortDirect) {
	*s = append(*s, PgSorter{fieldID: fieldId, direct: direct})
}

func (s PgSorters) Len() int {
	return len(s)
}

func (s PgSorters) SQL() string {
	if len(s) == 0 {
		return ""
	} else if len(s) == 1 {
		//most used cases
		return " ORDER BY " + s[0].SQL()
	}

	var sqlSt strings.Builder
	sqlSt.WriteString(" ORDER BY ")
	for i, sorter := range s {
		if i > 0 {
			sqlSt.WriteString(", ")
		}
		sqlSt.WriteString(sorter.SQL())
	}

	return sqlSt.String()
}
