package types

type DBUpdater interface {
	Model() DBModel
	Filter() DBFilters
	AddField(id string, value any)
}
