package types

type DBFilters interface {
	Add(
		pref string,
		fieldID string,
		value any, operator SQLFilterOperator,
		join FilterJoin,
	)
	AddFullTextSearch(pref, fieldID string, value any, join FilterJoin)
	AddArrayInclude(pref, fieldID string, value any, join FilterJoin)
	AddColumnArrayInclude(pref, fieldID string, value any, join FilterJoin)
	Len() int
}
