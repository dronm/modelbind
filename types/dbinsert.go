package types

type DBInserter interface {
	Model() DBModel
	AddField(id string, val any)
	AddRetField(id string, val any)
}
