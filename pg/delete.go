package pg

import (
	"fmt"

	"github.com/dronm/modelbind/types"
)

type PgDelete struct {
	policy  *boundPolicy
	request *boundPredicate
	model   types.DBModel
	filter  PgFilters
}

func NewPgDelete(model types.DBModel, filter PgFilters) PgDelete {
	return PgDelete{model: model, filter: filter}
}

func (d PgDelete) Model() types.DBModel {
	return d.model
}

func (d PgDelete) Filter() PgFilters {
	return d.filter
}

func (s PgDelete) SQL(queryParams *[]any) string {
	return mustSQL(s.BuildSQL(queryParams))
}

func (s PgDelete) BuildSQL(queryParams *[]any) (string, error) {
	return buildStatement(queryParams, func(params *[]any) (string, error) {
		if err := validateTarget(&s.filter, s.request, s.policy); err != nil {
			return "", err
		}
		filterSQL, err := scopedWhere(&s.filter, s.request, s.policy, params)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("DELETE FROM %s%s", s.model.Relation(), filterSQL), nil
	})
}
