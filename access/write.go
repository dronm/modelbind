package access

import (
	"fmt"
	"sort"
)

type WriteMode uint8

const (
	WriteInvalid WriteMode = iota
	WriteDefault
	WriteEnforced
	WriteAllowed
	WriteImmutable
)

// WriteRule is immutable. Construct rules with Default, Enforce, Allowed or
// Immutable. Only scalar values supported by NormalizeValue are accepted.
type WriteRule struct {
	mode   WriteMode
	value  any
	values []any
	err    error
}

// Default supplies an absent INSERT field. Explicit NULL is not absence.
func Default(value any) WriteRule {
	v, err := NormalizeValue(value)
	return WriteRule{mode: WriteDefault, value: v, err: err}
}

// Enforce supplies an absent INSERT field and rejects conflicting submitted
// values. On UPDATE an absent field is left unchanged, not silently repaired.
func Enforce(value any) WriteRule {
	v, err := NormalizeValue(value)
	return WriteRule{mode: WriteEnforced, value: v, err: err}
}

// Allowed requires a submitted INSERT value to be a member of the set. It does
// not choose an arbitrary default. On UPDATE absence is allowed; a submitted
// value must be a member. An empty set rejects every submitted value. NULL is
// accepted only when explicitly included in this set.
func Allowed(values ...any) WriteRule {
	r := WriteRule{mode: WriteAllowed, values: make([]any, len(values))}
	for i, value := range values {
		r.values[i], r.err = NormalizeValue(value)
		if r.err != nil {
			break
		}
	}
	return r
}

// Immutable rejects any submitted UPDATE of this field, even if the value might
// equal the current database value. It requires no pre-read and is UPDATE-only.
func Immutable() WriteRule { return WriteRule{mode: WriteImmutable} }

func (r WriteRule) Mode() WriteMode { return r.mode }

type WriteRules map[string]WriteRule

func (rules WriteRules) Fields() []string {
	fields := make([]string, 0, len(rules))
	for field := range rules {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

// Clone copies the rule map; individual rules are immutable scalar snapshots.
func (rules WriteRules) Clone() WriteRules {
	if rules == nil {
		return nil
	}
	result := make(WriteRules, len(rules))
	for field, rule := range rules {
		result[field] = rule
	}
	return result
}

func (rules WriteRules) Validate(operation Operation) error {
	if operation != Insert && operation != Update {
		return fmt.Errorf("%w: write rules require insert or update", ErrInvalidRule)
	}
	for _, field := range rules.Fields() {
		rule := rules[field]
		if !ValidField(field) {
			return &FieldError{Field: field, Code: "invalid_field", Kind: ErrInvalidField}
		}
		if rule.err != nil {
			return fmt.Errorf("%w: field %q: %w", ErrInvalidRule, field, rule.err)
		}
		switch rule.mode {
		case WriteDefault:
			if operation != Insert {
				return &FieldError{Field: field, Code: "default_requires_insert", Kind: ErrInvalidRule}
			}
		case WriteImmutable:
			if operation != Update {
				return &FieldError{Field: field, Code: "immutable_requires_update", Kind: ErrInvalidRule}
			}
		case WriteEnforced, WriteAllowed:
		default:
			return &FieldError{Field: field, Code: "missing_rule", Kind: ErrInvalidRule}
		}
	}
	return nil
}

// Apply produces effective values without modifying submitted. Map membership
// denotes presence; a present nil is an explicit NULL. Values unrelated to a
// rule are copied unchanged. Only governed scalar values are normalized.
func (rules WriteRules) Apply(operation Operation, submitted map[string]any) (map[string]any, error) {
	if err := rules.Validate(operation); err != nil {
		return nil, err
	}
	result := make(map[string]any, len(submitted)+len(rules))
	for field, value := range submitted {
		result[field] = value
	}
	for _, field := range rules.Fields() {
		rule := rules[field]
		value, present := submitted[field]
		if !present {
			if operation == Update {
				continue
			}
			switch rule.mode {
			case WriteDefault, WriteEnforced:
				result[field] = rule.value
			case WriteAllowed:
				return nil, &FieldError{Field: field, Code: "value_required", Kind: ErrWriteConflict}
			}
			continue
		}
		if rule.mode == WriteImmutable {
			return nil, &FieldError{Field: field, Code: "immutable_field", Kind: ErrWriteConflict}
		}
		normalized, err := NormalizeValue(value)
		if err != nil {
			return nil, fmt.Errorf("%w: field %q: %w", ErrWriteConflict, field, err)
		}
		switch rule.mode {
		case WriteEnforced:
			equal, err := ValuesEqual(normalized, rule.value)
			if err != nil || !equal {
				return nil, &FieldError{Field: field, Code: "enforced_value", Kind: ErrWriteConflict}
			}
		case WriteAllowed:
			allowed := false
			for _, candidate := range rule.values {
				equal, _ := ValuesEqual(normalized, candidate)
				if equal {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, &FieldError{Field: field, Code: "value_not_allowed", Kind: ErrWriteConflict}
			}
		}
		result[field] = normalized
	}
	return result, nil
}
