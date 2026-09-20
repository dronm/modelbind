package access

import (
	"fmt"
	"regexp"
	"sort"
)

// Kind is the closed set of supported predicate nodes. There is deliberately no
// raw-SQL node. The zero Predicate is invalid, not unrestricted access.
type Kind uint8

const (
	KindInvalid Kind = iota
	KindTrue
	KindFalse
	KindCompare
	KindIn
	KindIsNull
	KindIsNotNull
	KindAnd
	KindOr
)

type Operator string

const (
	OpEqual          Operator = "="
	OpNotEqual       Operator = "<>"
	OpLess           Operator = "<"
	OpLessOrEqual    Operator = "<="
	OpGreater        Operator = ">"
	OpGreaterOrEqual Operator = ">="
)

var fieldRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidField reports whether a logical field ID is valid. SQL aliases and JSON
// paths belong in the database adapter's explicit field map, not in this ID.
func ValidField(field string) bool { return fieldRE.MatchString(field) }

// Predicate is immutable through its public API. Constructors snapshot scalar
// parameters so a caller cannot later mutate a policy through a pointer/slice.
// Constructor errors are reported by Validate and by the PostgreSQL compiler.
type Predicate struct {
	kind     Kind
	field    string
	op       Operator
	values   []any
	children []Predicate
	err      error
}

func True() Predicate                               { return Predicate{kind: KindTrue} }
func False() Predicate                              { return Predicate{kind: KindFalse} }
func Eq(field string, value any) Predicate          { return Compare(field, OpEqual, value) }
func NotEq(field string, value any) Predicate       { return Compare(field, OpNotEqual, value) }
func Less(field string, value any) Predicate        { return Compare(field, OpLess, value) }
func LessOrEqual(field string, value any) Predicate { return Compare(field, OpLessOrEqual, value) }
func Greater(field string, value any) Predicate     { return Compare(field, OpGreater, value) }
func GreaterOrEqual(field string, value any) Predicate {
	return Compare(field, OpGreaterOrEqual, value)
}
func IsNull(field string) Predicate    { return Predicate{kind: KindIsNull, field: field} }
func IsNotNull(field string) Predicate { return Predicate{kind: KindIsNotNull, field: field} }

func Compare(field string, op Operator, value any) Predicate {
	v, err := NormalizeValue(value)
	if err == nil && v == nil {
		switch op {
		case OpEqual:
			return IsNull(field)
		case OpNotEqual:
			return IsNotNull(field)
		default:
			err = fmt.Errorf("%w: NULL requires equality or an explicit null predicate", ErrInvalidPredicate)
		}
	}
	return Predicate{kind: KindCompare, field: field, op: op, values: []any{v}, err: err}
}

// In is membership in a set of non-null scalars. An empty set compiles to FALSE,
// never to an omitted predicate. Use Or(In(...), IsNull(...)) to permit SQL NULL.
func In(field string, values ...any) Predicate {
	p := Predicate{kind: KindIn, field: field, values: make([]any, len(values))}
	for i, value := range values {
		v, err := NormalizeValue(value)
		if err != nil {
			p.err = err
			break
		}
		if v == nil {
			p.err = fmt.Errorf("%w: IN does not accept NULL; use IsNull explicitly", ErrInvalidPredicate)
			break
		}
		p.values[i] = v
	}
	return p
}

// And and Or require at least one valid child. Use True/False for an intentional
// constant instead of relying on an empty or accidentally missing restriction.
func And(children ...Predicate) Predicate {
	return Predicate{kind: KindAnd, children: append([]Predicate(nil), children...)}
}
func Or(children ...Predicate) Predicate {
	return Predicate{kind: KindOr, children: append([]Predicate(nil), children...)}
}

func (p Predicate) Kind() Kind            { return p.kind }
func (p Predicate) Field() string         { return p.field }
func (p Predicate) Operator() Operator    { return p.op }
func (p Predicate) Values() []any         { return append([]any(nil), p.values...) }
func (p Predicate) Children() []Predicate { return append([]Predicate(nil), p.children...) }
func (p Predicate) IsZero() bool          { return p.kind == KindInvalid && p.err == nil }

func (p Predicate) Validate() error { return p.validate(0) }

func (p Predicate) validate(depth int) error {
	if depth > 64 {
		return fmt.Errorf("%w: predicate nesting exceeds 64 levels", ErrInvalidPredicate)
	}
	if p.err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidPredicate, p.err)
	}
	switch p.kind {
	case KindTrue, KindFalse:
		return nil
	case KindAnd, KindOr:
		if len(p.children) == 0 {
			return fmt.Errorf("%w: empty predicate group", ErrInvalidPredicate)
		}
		for _, child := range p.children {
			if err := child.validate(depth + 1); err != nil {
				return err
			}
		}
		return nil
	case KindCompare:
		switch p.op {
		case OpEqual, OpNotEqual, OpLess, OpLessOrEqual, OpGreater, OpGreaterOrEqual:
		default:
			return fmt.Errorf("%w: unsupported comparison operator", ErrInvalidPredicate)
		}
	case KindIn, KindIsNull, KindIsNotNull:
	default:
		return fmt.Errorf("%w: missing predicate", ErrInvalidPredicate)
	}
	if !ValidField(p.field) {
		return &FieldError{Field: p.field, Code: "invalid_field", Kind: ErrInvalidField}
	}
	return nil
}

// Fields returns the logical fields referenced by a valid predicate, in sorted
// order. Validation must succeed first; invalid predicates return no fields.
func (p Predicate) Fields() []string {
	if p.Validate() != nil {
		return nil
	}
	set := make(map[string]struct{})
	var visit func(Predicate)
	visit = func(node Predicate) {
		if node.field != "" {
			set[node.field] = struct{}{}
		}
		for _, child := range node.children {
			visit(child)
		}
	}
	visit(p)
	fields := make([]string, 0, len(set))
	for field := range set {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}
