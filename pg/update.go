package pg

import (
	"fmt"

	"github.com/dronm/modelbind/types"
)

type PgUpdate struct {
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
	var assignerSQL string
	if u.assigner != nil {
		assignerSQL = u.assigner.SQL(queryParams)
	}
	var filterSQL string
	if u.filter != nil {
		filterSQL = u.filter.SQL(queryParams)
	}
	var limitSQL string
	if u.limit != nil {
		limitSQL = u.limit.SQL()
	}

	return fmt.Sprintf("UPDATE %s SET %s%s%s",
		u.model.Relation(),
		assignerSQL,
		filterSQL,
		limitSQL,
	)
}
