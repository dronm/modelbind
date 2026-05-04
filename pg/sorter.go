package pg

import (
	"fmt"

	"github.com/dronm/modelbind/types"
)

type PgSorter struct {
	fieldID   string
	direct    types.SQLSortDirect
	fieldPref string
}

func (s PgSorter) FieldID() string {
	return s.fieldID
}

func (s PgSorter) Direct() types.SQLSortDirect {
	return s.direct
}

func (s PgSorter) FieldPref() string {
	return s.fieldPref
}

func (s PgSorter) SQL() string {
	fieldID := s.fieldID
	if s.fieldPref != "" {
		fieldID = s.fieldPref + "." + fieldID
	}

	safeFieldID, err := sanitizeSQLFieldRef(fieldID)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s %s", safeFieldID, s.direct)
}
