package pg

import (
	"fmt"
	"strings"

	"github.com/dronm/modelbind/types"
)

type PgField struct {
	ID    string
	Value any
}

type PgInsert struct {
	model          types.DBModel
	values         []any
	fields         []PgField
	retFieldIds    []string
	retFieldValues []any
}

func NewPgInsert(model types.DBModel) *PgInsert {
	return &PgInsert{model: model}
}

func (s PgInsert) Model() types.DBModel {
	return s.model
}

func (s *PgInsert) AddRetField(id string, val any) {
	s.retFieldIds = append(s.retFieldIds, id)
	s.retFieldValues = append(s.retFieldValues, val)
}

func (s PgInsert) RetFieldIds() []string {
	return s.retFieldIds
}

func (s PgInsert) RetFieldValues() []any {
	return s.retFieldValues
}

func (s PgInsert) RetFields() map[string]any {
	res := make(map[string]any, len(s.retFieldIds))
	for i, f := range s.retFieldIds {
		res[f] = s.retFieldValues[i]
	}
	return res
}

func (s *PgInsert) AddField(fieldId string, val any) {
	s.fields = append(s.fields, PgField{ID: fieldId, Value: val})
}

func (s PgInsert) InsertFieldLen() int {
	return len(s.fields)
}

func (s PgInsert) SQL(queryParams *[]any) string {
	paramInd := len(*queryParams)
	var fieldIds strings.Builder
	var fieldVals strings.Builder
	for _, field := range s.fields {
		if fieldIds.Len() > 0 {
			fieldIds.WriteString(",")
			fieldVals.WriteString(",")
		}
		paramInd++
		fieldVals.WriteString(fmt.Sprintf("$%d", paramInd))
		safeFieldID, err := sanitizeSQLFieldRef(field.ID)
		if err != nil {
			panic(err)
		}
		fieldIds.WriteString(safeFieldID)
		*queryParams = append(*queryParams, field.Value)
	}

	retFields := ""
	if len(s.retFieldIds) > 0 {
		retFields = " RETURNING " + joinSafeFieldRefs(s.retFieldIds)
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)%s",
		s.model.Relation(),
		fieldIds.String(),
		fieldVals.String(),
		retFields,
	)
}
