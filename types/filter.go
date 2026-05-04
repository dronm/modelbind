package types

type SQLFilterOperator string

const (
	SQLFilterOperatorEq       SQLFilterOperator = "="
	SQLFilterOperatorLess     SQLFilterOperator = "<"
	SQLFilterOperatorGr       SQLFilterOperator = ">"
	SQLFilterOperatorLessEq   SQLFilterOperator = "<="
	SQLFilterOperatorGrEq     SQLFilterOperator = ">="
	SQLFilterOperatorLk       SQLFilterOperator = "LIKE"
	SQLFilterOperatorILk      SQLFilterOperator = "ILIKE"
	SQLFilterOperatorNotEq    SQLFilterOperator = "<>"
	SQLFilterOperatorIs       SQLFilterOperator = "IS"
	SQLFilterOperatorIn       SQLFilterOperator = "IS NOT"
	SQLFilterOperatorIncl     SQLFilterOperator = "IN"
	SQLFilterOperatorAny      SQLFilterOperator = "ANY"
	SQLFilterOperatorHas      SQLFilterOperator = "ANY"
	SQLFilterOperatorOverlap  SQLFilterOperator = "&&"
	SQLFilterOperatorContains SQLFilterOperator = "@>"
	SQLFilterOperatorTS       SQLFilterOperator = "@@"
)

type FilterJoin string

const (
	SQLFilterJoinAnd FilterJoin = "AND"
	SQLFilterJoinOr  FilterJoin = "OR"
)

type DBFilter interface {
	FieldID() string
	Value() any
	Operator() SQLFilterOperator
	Expression() string // validated,sanatized expression
	Join() FilterJoin
	FieldPref() string
	SQL() string
}
