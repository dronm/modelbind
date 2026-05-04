package pg

import (
	"fmt"
	"strings"

	"github.com/dronm/modelbind/types"
)

const exprParam = "{{PARAM}}"

type PgFilter struct {
	fieldID    string
	value      any
	operator   types.SQLFilterOperator
	expression string
	join       types.FilterJoin
	fieldPref  string //table, table alias TABLE.fieldID
}

func NewPgFilter(fieldID string, value any) *PgFilter {
	return &PgFilter{
		fieldID: fieldID, value: value,
		operator: types.SQLFilterOperatorEq,
		join:     types.SQLFilterJoinAnd,
	}
}

func (f PgFilter) FieldID() string {
	return f.fieldID
}

func (f PgFilter) Value() any {
	return f.value
}

func (f *PgFilter) SetValue(value any) {
	f.value = value
}

func (f PgFilter) Operator() types.SQLFilterOperator {
	return f.operator
}

func (f *PgFilter) SetOperator(op types.SQLFilterOperator) {
	f.operator = op
}

func (f PgFilter) Expression() string {
	return f.expression
}

func (f *PgFilter) SetExpression(expr string) {
	f.expression = expr
}

func (f PgFilter) Join() types.FilterJoin {
	return f.join
}

func (f *PgFilter) SetJoin(j types.FilterJoin) {
	f.join = j
}

func (f PgFilter) FieldPref() string {
	return f.fieldPref
}

func (f PgFilter) SQL(queryParams *[]any) string {
	fieldID := f.fieldID
	if f.fieldPref != "" {
		fieldID = f.fieldPref + "." + fieldID
	}
	if fieldID != "" {
		safeFieldID, err := sanitizeSQLFieldRef(fieldID)
		if err != nil {
			panic(err)
		}
		fieldID = safeFieldID
	}

	if f.expression != "" {
		if strings.Contains(f.expression, exprParam) {
			parInd := 0
			if queryParams != nil {
				parInd = len(*queryParams)
			}
			parInd++
			*queryParams = append(*queryParams, f.value)
			return strings.Replace(f.expression, exprParam, fmt.Sprintf("$%d", parInd), 1)
		}

		return f.expression
	}

	if f.value == nil && (f.operator == types.SQLFilterOperatorIs || f.operator == types.SQLFilterOperatorIn) {
		return fmt.Sprintf("%s %s NULL", fieldID, f.operator)
	}

	if f.operator == types.SQLFilterOperatorAny {
		parInd := 0
		if queryParams != nil {
			parInd = len(*queryParams)
		}
		parInd++
		*queryParams = append(*queryParams, f.value)
		return fmt.Sprintf("%s = ANY($%d)", fieldID, parInd)
	}

	if f.operator == "" {
		f.operator = types.SQLFilterOperatorEq
	}

	parInd := 0
	if queryParams != nil {
		parInd = len(*queryParams)
	}
	parInd++
	*queryParams = append(*queryParams, f.value)

	return fmt.Sprintf("%s %s $%d", fieldID, f.operator, parInd)
}
