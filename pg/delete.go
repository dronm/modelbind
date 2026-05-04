package pg

import (
	"fmt"

	"github.com/dronm/modelbind/types"
)

type PgDelete struct {
	model  types.DBModel
	filter PgFilters
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
	return fmt.Sprintf("DELETE FROM %s%s",
		s.model.Relation(),
		s.filter.SQL(queryParams),
	)
}
