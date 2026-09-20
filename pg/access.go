package pg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dronm/modelbind/access"
)

var (
	ErrMissingTarget  = errors.New("protected update/delete requires a caller target predicate")
	ErrNoAssignments  = errors.New("protected update has no assignments")
	ErrDuplicateField = errors.New("duplicate write field")
)

// PolicyBinding maps row predicates and write rules independently. Rows may use
// aliases, while Writes must use unqualified writable column names. This lets a
// projection and a base-table model share one logical policy resource.
type PolicyBinding struct {
	Rows   FieldMap
	Writes FieldMap
}

type boundPolicy struct {
	policy    access.Policy
	operation access.Operation
	rows      *boundPredicate
	writes    FieldMap
	err       error
}

func bindPolicy(policy access.Policy, operation access.Operation, binding PolicyBinding) *boundPolicy {
	b := &boundPolicy{policy: policy.Clone(), operation: operation, writes: binding.Writes.clone()}
	if b.err = policy.Validate(operation); b.err != nil {
		return b
	}
	if policy.Decision == access.Denied {
		b.err = access.ErrDenied
		return b
	}
	if !policy.Rows.IsZero() {
		b.rows = bindPredicate(policy.Rows, binding.Rows)
		if b.err = b.rows.err; b.err != nil {
			return b
		}
	}
	used := make(map[string]string)
	for _, field := range policy.Writes.Fields() {
		column, ok := b.writes[field]
		if !ok {
			b.err = &access.FieldError{Field: field, Code: "missing_write_mapping", Kind: access.ErrInvalidField}
			return b
		}
		safe, err := writeColumn(column)
		if err != nil {
			b.err = fmt.Errorf("%w: field %q: %w", access.ErrInvalidField, field, err)
			return b
		}
		canonical := strings.ToLower(safe)
		if previous, duplicate := used[canonical]; duplicate {
			b.err = fmt.Errorf("%w: logical fields %q and %q share a write column", access.ErrInvalidField, previous, field)
			return b
		}
		used[canonical] = field
		b.writes[field] = safe
	}
	return b
}

func writeColumn(column string) (string, error) {
	column = strings.TrimSpace(column)
	if !identRE.MatchString(column) {
		return "", fmt.Errorf("%w: expected an unqualified write column", access.ErrInvalidField)
	}
	return column, nil
}

func (p *boundPolicy) check() error {
	if p != nil {
		return p.err
	}
	return nil
}

func validateTarget(filter *PgFilters, request *boundPredicate, policy *boundPolicy) error {
	if err := policy.check(); err != nil {
		return err
	}
	if request != nil && request.err != nil {
		return request.err
	}
	if policy == nil {
		return nil // Preserve legacy, opt-in behavior.
	}
	if filter != nil && filter.Len() > 0 {
		return nil
	}
	if request != nil && targetRestricted(request.predicate) {
		return nil
	}
	return ErrMissingTarget
}

func scopedWhere(filter *PgFilters, request *boundPredicate, policy *boundPolicy, params *[]any) (string, error) {
	if err := policy.check(); err != nil {
		return "", err
	}
	if request == nil && policy == nil {
		if filter != nil {
			return filter.SQL(params), nil
		}
		return "", nil
	}
	parts := make([]string, 0, 3)
	if filter != nil && filter.Len() > 0 {
		parts = append(parts, "("+strings.TrimPrefix(filter.SQL(params), " WHERE ")+")")
	}
	if request != nil {
		expr, err := request.sql(params)
		if err != nil {
			return "", err
		}
		parts = append(parts, "("+expr+")")
	}
	if policy != nil && policy.rows != nil {
		expr, err := policy.rows.sql(params)
		if err != nil {
			return "", err
		}
		parts = append(parts, "("+expr+")")
	}
	if len(parts) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(parts, " AND "), nil
}

// effectiveFields rechecks rules at build time. Installing a policy and then
// calling AddField/SetField cannot override it. No caller input is mutated.
func effectiveFields(fields []PgField, policy *boundPolicy) ([]PgField, error) {
	if err := policy.check(); err != nil {
		return nil, err
	}
	if policy == nil {
		return fields, nil
	}
	result := append([]PgField(nil), fields...)
	indices := make(map[string]int, len(fields))
	for i, field := range result {
		column, err := writeColumn(field.ID)
		if err != nil {
			return nil, err
		}
		canonical := strings.ToLower(column)
		if _, duplicate := indices[canonical]; duplicate {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateField, column)
		}
		result[i].ID = column
		indices[canonical] = i
	}
	submitted := make(map[string]any, len(policy.policy.Writes))
	for field := range policy.policy.Writes {
		if index, present := indices[strings.ToLower(policy.writes[field])]; present {
			submitted[field] = result[index].Value
		}
	}
	effective, err := policy.policy.Writes.Apply(policy.operation, submitted)
	if err != nil {
		return nil, err
	}
	for _, field := range policy.policy.Writes.Fields() {
		value, present := effective[field]
		if !present {
			continue
		}
		column := policy.writes[field]
		if index, exists := indices[strings.ToLower(column)]; exists {
			result[index].Value = value
		} else {
			result = append(result, PgField{ID: column, Value: value})
		}
	}
	return result, nil
}

// buildStatement makes parameter allocation transactional. Existing SQL methods
// historically panic on unsafe identifiers; new BuildSQL methods expose those
// errors without returning a partial query or partially appended arguments.
func buildStatement(params *[]any, build func(*[]any) (string, error)) (sql string, err error) {
	if params == nil {
		return "", errors.New("pg: query parameter slice pointer is nil")
	}
	temporary := append([]any(nil), (*params)...)
	defer func() {
		if recovered := recover(); recovered != nil {
			sql = ""
			if cause, ok := recovered.(error); ok {
				err = fmt.Errorf("pg: cannot build SQL: %w", cause)
			} else {
				err = fmt.Errorf("pg: cannot build SQL: %v", recovered)
			}
		}
	}()
	sql, err = build(&temporary)
	if err != nil {
		return "", err
	}
	*params = temporary
	return sql, nil
}

func mustSQL(sql string, err error) string {
	if err != nil {
		panic(err)
	}
	return sql
}
