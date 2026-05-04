package modelbind

// This file provides types and functions for parsing
// collection parameters from client queries.

// FilterOperatorParam is a clietn query operator.
type FilterOperatorParam string

// client query fileter operator values
const (
	FilterOperParEq     FilterOperatorParam = "e"    // equal
	FilterOperParLess   FilterOperatorParam = "l"    // less
	FilterOperParGr     FilterOperatorParam = "g"    // greater
	FilterOperParLessEq FilterOperatorParam = "le"   // less and equal
	FilterOperParGrEq   FilterOperatorParam = "ge"   // greater and equal
	FilterOperParLk     FilterOperatorParam = "lk"   // like
	FilterOperParILk    FilterOperatorParam = "ilk"  // ilike
	FilterOperParNotEq  FilterOperatorParam = "ne"   // not equal
	FilterOperParIs     FilterOperatorParam = "i"    // IS
	FilterOperParIn     FilterOperatorParam = "in"   // IS NOT
	FilterOperParIncl   FilterOperatorParam = "incl" // include: column IN (param_array)
	FilterOperParAny    FilterOperatorParam = "any"  // Any: column = ANY(param array)

	FilterOperParHas FilterOperatorParam = "has" // Any: param = ANY(array_column)

	// overlap: any element in the column matches any element in the parameter
	FilterOperParOverlap FilterOperatorParam = "overlap" // array_column && param_array

	// Find rows where the parameter array is a subset of the column array
	FilterOperParContains FilterOperatorParam = "contains" // contains @> param_array

	FilterOperParTS FilterOperatorParam = "fts" // full text search
)

type FilterJoinParam string

const (
	FilterParJoinAnd FilterJoinParam = "and"
	FilterParJoinOr  FilterJoinParam = "or"
)

type SortParam string

// client query sorting values
const (
	SortParAsc  SortParam = "a" // asc
	SortParDesc SortParam = "d" // desc
)

type CollectionSorter struct {
	Field  string    `json:"f"`
	Direct SortParam `json:"d"`
}

type CollectionFilterField struct {
	Operator FilterOperatorParam `json:"o"`
	Value    any                 `json:"v"`
}

type CollectionFilter struct {
	Join   FilterJoinParam                  `json:"j"`
	Fields map[string]CollectionFilterField `json:"f"`
}

type (
	CollectionFrom  int
	CollectionCount int
)

// CollectionParams holds all unmarshaled
// client query parameters.
type CollectionParams struct {
	Filter []CollectionFilter `json:"filter"`
	Sorter []CollectionSorter `json:"sorter"`
	From   CollectionFrom     `json:"from"`
	Count  CollectionCount    `json:"count"`
}
