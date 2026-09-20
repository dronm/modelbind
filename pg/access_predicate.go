package pg

import (
	"fmt"
	"strings"

	"github.com/dronm/modelbind/access"
)

// FieldMap explicitly binds logical policy field IDs to trusted SQL column
// references. There is no fallback to the logical name. Values are validated;
// aliases and supported JSON paths are allowed for row predicates.
type FieldMap map[string]string

func (fields FieldMap) clone() FieldMap {
	result := make(FieldMap, len(fields))
	for field, column := range fields {
		result[field] = column
	}
	return result
}

type boundPredicate struct {
	predicate access.Predicate
	fields    FieldMap
	err       error
}

func bindPredicate(predicate access.Predicate, fields FieldMap) *boundPredicate {
	bound := &boundPredicate{predicate: predicate, fields: fields.clone()}
	if err := predicate.Validate(); err != nil {
		bound.err = err
		return bound
	}
	for field, column := range bound.fields {
		if !access.ValidField(field) {
			bound.err = &access.FieldError{Field: field, Code: "invalid_field", Kind: access.ErrInvalidField}
			return bound
		}
		safe, err := sanitizeSQLFieldRef(column)
		if err != nil {
			bound.err = fmt.Errorf("%w: field %q: %w", access.ErrInvalidField, field, err)
			return bound
		}
		bound.fields[field] = safe
	}
	for _, field := range predicate.Fields() {
		if _, ok := bound.fields[field]; !ok {
			bound.err = &access.FieldError{Field: field, Code: "missing_row_mapping", Kind: access.ErrInvalidField}
			return bound
		}
	}
	return bound
}

// CompilePredicate returns an SQL expression (without WHERE) and appends its
// parameters after any existing ones. On error parameters remain unchanged.
// Policy values never become SQL text. Empty IN sets compile to FALSE.
func CompilePredicate(predicate access.Predicate, fields FieldMap, queryParams *[]any) (string, error) {
	bound := bindPredicate(predicate, fields)
	return buildStatement(queryParams, func(params *[]any) (string, error) {
		return bound.sql(params)
	})
}

func (b *boundPredicate) sql(params *[]any) (string, error) {
	if b == nil {
		return "", nil
	}
	if b.err != nil {
		return "", b.err
	}
	return compilePredicateNode(b.predicate, b.fields, params), nil
}

func compilePredicateNode(p access.Predicate, fields FieldMap, params *[]any) string {
	parameter := func(value any) string {
		*params = append(*params, value)
		return fmt.Sprintf("$%d", len(*params))
	}
	switch p.Kind() {
	case access.KindTrue:
		return "TRUE"
	case access.KindFalse:
		return "FALSE"
	case access.KindCompare:
		return fields[p.Field()] + " " + string(p.Operator()) + " " + parameter(p.Values()[0])
	case access.KindIsNull:
		return fields[p.Field()] + " IS NULL"
	case access.KindIsNotNull:
		return fields[p.Field()] + " IS NOT NULL"
	case access.KindIn:
		values := p.Values()
		if len(values) == 0 {
			return "FALSE"
		}
		placeholders := make([]string, len(values))
		for i, value := range values {
			placeholders[i] = parameter(value)
		}
		return fields[p.Field()] + " IN (" + strings.Join(placeholders, ",") + ")"
	case access.KindAnd, access.KindOr:
		children := p.Children()
		parts := make([]string, len(children))
		for i, child := range children {
			parts[i] = "(" + compilePredicateNode(child, fields, params) + ")"
		}
		join := " AND "
		if p.Kind() == access.KindOr {
			join = " OR "
		}
		return strings.Join(parts, join)
	default:
		// Only validated, closed Predicate nodes can reach this compiler.
		panic(access.ErrInvalidPredicate)
	}
}

// targetRestricted is a syntactic safeguard, not proof of a primary-key lookup.
// In particular a caller's explicit TRUE (including OR TRUE) is not a target.
func targetRestricted(p access.Predicate) bool {
	switch p.Kind() {
	case access.KindInvalid, access.KindTrue:
		return false
	case access.KindAnd:
		for _, child := range p.Children() {
			if targetRestricted(child) {
				return true
			}
		}
		return false
	case access.KindOr:
		for _, child := range p.Children() {
			if !targetRestricted(child) {
				return false
			}
		}
		return len(p.Children()) > 0
	default:
		return true
	}
}
