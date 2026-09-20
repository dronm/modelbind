package pg

import (
	"fmt"

	"github.com/dronm/modelbind/types"
)

type PgUpdate struct {
	policy   *boundPolicy
	request  *boundPredicate
	model    types.DBModel
	assigner *PgAssigners
	filter   *PgFilters
	limit    *PgLimit
}

func NewPgUpdate(model types.DBModel) *PgUpdate {
	return &PgUpdate{model: model, filter: &PgFilters{}, assigner: &PgAssigners{}}
}

func (u PgUpdate) Model() types.DBModel {
	return u.model
}

func (u *PgUpdate) AddField(id string, value any) {
	u.assigner.Add(id, value)
}

func (u PgUpdate) AssignerLen() int {
	if u.assigner == nil {
		return 0
	}
	return u.assigner.Len()
}

func (u PgUpdate) Filter() types.DBFilters {
	return u.filter
}

func (u PgUpdate) SQL(queryParams *[]any) string {
	return mustSQL(u.BuildSQL(queryParams))
}

func (u PgUpdate) BuildSQL(queryParams *[]any) (string, error) {
	return buildStatement(queryParams, func(params *[]any) (string, error) {
		if err := validateTarget(u.filter, u.request, u.policy); err != nil {
			return "", err
		}
		var fields []PgField
		if u.assigner != nil {
			for _, assignment := range *u.assigner {
				fields = append(fields, PgField{ID: assignment.fieldID, Value: assignment.value})
			}
		}
		fields, err := effectiveFields(fields, u.policy)
		if err != nil {
			return "", err
		}
		if u.policy != nil && len(fields) == 0 {
			return "", ErrNoAssignments
		}
		assigner := PgAssigners{}
		for _, field := range fields {
			assigner.Add(field.ID, field.Value)
		}
		assignerSQL := assigner.SQL(params)
		filterSQL, err := scopedWhere(u.filter, u.request, u.policy, params)
		if err != nil {
			return "", err
		}
		var limitSQL string
		if u.limit != nil {
			limitSQL = u.limit.SQL()
		}
		return fmt.Sprintf("UPDATE %s SET %s%s%s", u.model.Relation(), assignerSQL, filterSQL, limitSQL), nil
	})
}

// SetField replaces a column's assignment in place, removing any duplicates.
// Access rules are still rechecked when BuildSQL/SQL is called.
func (u *PgUpdate) SetField(id string, value any) error {
	if u.assigner == nil {
		u.assigner = &PgAssigners{}
	}
	return u.assigner.Set(id, value)
}
