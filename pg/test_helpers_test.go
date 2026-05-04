package pg

import "github.com/dronm/modelbind/types"

type testModel struct {
	relation string
}

func (m testModel) Relation() string {
	return m.relation
}

func (m testModel) CollectionAgg() any {
	return nil
}

type wrongFilters struct{}

func (wrongFilters) Add(string, string, any, types.SQLFilterOperator, types.FilterJoin) {}
func (wrongFilters) AddFullTextSearch(string, string, any, types.FilterJoin)            {}
func (wrongFilters) AddArrayInclude(string, string, any, types.FilterJoin)              {}
func (wrongFilters) AddColumnArrayInclude(string, string, any, types.FilterJoin)        {}
func (wrongFilters) Len() int                                                           { return 0 }
