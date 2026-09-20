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
	policy         *boundPolicy
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
	return mustSQL(s.BuildSQL(queryParams))
}

func (s PgInsert) BuildSQL(queryParams *[]any) (string, error) {
	return buildStatement(queryParams, func(params *[]any) (string, error) {
		fields, err := effectiveFields(s.fields, s.policy)
		if err != nil {
			return "", err
		}
		var fieldIDs, fieldValues []string
		for _, field := range fields {
			safe, err := sanitizeSQLFieldRef(field.ID)
			if err != nil {
				return "", err
			}
			fieldIDs = append(fieldIDs, safe)
			*params = append(*params, field.Value)
			fieldValues = append(fieldValues, fmt.Sprintf("$%d", len(*params)))
		}
		returning := ""
		if len(s.retFieldIds) > 0 {
			returning = " RETURNING " + joinSafeFieldRefs(s.retFieldIds)
		}
		if s.policy != nil && len(fields) == 0 {
			return "INSERT INTO " + s.model.Relation() + " DEFAULT VALUES" + returning, nil
		}
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)%s",
			s.model.Relation(), strings.Join(fieldIDs, ","), strings.Join(fieldValues, ","), returning), nil
	})
}

// SetField replaces an existing column in place and removes duplicate entries.
// AddField retains its legacy append semantics. Policy checks run at build time.
func (s *PgInsert) SetField(id string, value any) error {
	column, err := writeColumn(id)
	if err != nil {
		return err
	}
	result := make([]PgField, 0, len(s.fields)+1)
	replaced := false
	for _, field := range s.fields {
		if strings.EqualFold(strings.TrimSpace(field.ID), column) {
			if !replaced {
				result = append(result, PgField{ID: column, Value: value})
				replaced = true
			}
			continue
		}
		result = append(result, field)
	}
	if !replaced {
		result = append(result, PgField{ID: column, Value: value})
	}
	s.fields = result
	return nil
}
