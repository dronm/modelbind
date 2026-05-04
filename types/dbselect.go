package types

type DBModel interface {
	Relation() string
}

type DBAggModel interface {
	DBModel
	CollectionAgg() any // structure with agg:"count(*)"
}

type PrepareModel interface {
	AddField(id string, val any)
}

// DBDetailSelecter is for detail model.
type DBDetailSelecter interface {
	Model() DBModel
	Filter() DBFilters
	SetFilter(DBFilters) error
	AddField(id string, val any)
}

// DBSelecter is for list model.
type DBSelecter interface {
	Model() DBAggModel
	Filter() DBFilters
	SetFilter(DBFilters) error
	Limit() DBLimit
	Sorter() DBSorters
	AddField(id string, val any)
	AddAggField(aggFn string, val any)
}
