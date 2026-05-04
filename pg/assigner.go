package pg

import (
	"fmt"
)

type PgAssigner struct {
	fieldID string
	value   any
}

func (a PgAssigner) FieldID() string {
	return a.fieldID
}

func (a PgAssigner) SQL(queryParams *[]any) string {
	parInd := 0
	if queryParams != nil {
		parInd = len(*queryParams)
	}
	parInd++

	*queryParams = append(*queryParams, a.value)

	safeFieldID, err := sanitizeSQLFieldRef(a.fieldID)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s = $%d", safeFieldID, parInd)
}
