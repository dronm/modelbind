package access

import "fmt"

type Operation string

const (
	Read   Operation = "read"
	Insert Operation = "insert"
	Update Operation = "update"
	Delete Operation = "delete"
)

type Decision uint8

const (
	DecisionInvalid Decision = iota
	Unrestricted
	Restricted
	Denied
)

// Policy is a resolved policy for one operation. No zero-value policy is valid.
// Rows determines access to existing rows, independently of Writes. In
// particular an UPDATE row predicate is NOT automatically a new-row check:
// pair ownership predicates with Enforce/Allowed/Immutable write rules.
// INSERT uses Writes only and rejects Rows, rather than guessing assignments.
type Policy struct {
	Decision Decision
	Rows     Predicate
	Writes   WriteRules
}

func AllowAll() Policy                   { return Policy{Decision: Unrestricted} }
func DenyAll() Policy                    { return Policy{Decision: Denied} }
func Scope(rows Predicate) Policy        { return Policy{Decision: Restricted, Rows: rows} }
func ForInsert(writes WriteRules) Policy { return Policy{Decision: Restricted, Writes: writes.Clone()} }

// Validate checks policy structure, not permission. DenyAll is structurally
// valid; adapters must reject it with ErrDenied before producing a query.
func (p Policy) Validate(operation Operation) error {
	switch operation {
	case Read, Insert, Update, Delete:
	default:
		return fmt.Errorf("%w: unknown operation", ErrInvalidPolicy)
	}
	switch p.Decision {
	case Unrestricted, Denied:
		if !p.Rows.IsZero() || len(p.Writes) != 0 {
			return fmt.Errorf("%w: unrestricted/denied policies cannot contain restrictions", ErrInvalidPolicy)
		}
		return nil
	case Restricted:
	default:
		return fmt.Errorf("%w: explicit decision required", ErrInvalidPolicy)
	}
	if operation == Insert {
		if !p.Rows.IsZero() {
			return fmt.Errorf("%w: insert requires write rules, not an existing-row predicate", ErrInvalidPolicy)
		}
		if len(p.Writes) == 0 {
			return fmt.Errorf("%w: restricted insert requires write rules", ErrInvalidPolicy)
		}
	} else if err := p.Rows.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidPolicy, err)
	}
	if operation == Read || operation == Delete {
		if len(p.Writes) != 0 {
			return fmt.Errorf("%w: read/delete cannot contain write rules", ErrInvalidPolicy)
		}
	} else if err := p.Writes.Validate(operation); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidPolicy, err)
	}
	return nil
}

func (p Policy) Clone() Policy {
	p.Writes = p.Writes.Clone()
	return p
}
