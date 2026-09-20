package pg

import "github.com/dronm/modelbind/access"

// SetAccessPolicy attaches a resolved read policy. Even a failed call is
// remembered: BuildSQL returns the error and SQL panics, rather than silently
// generating an unprotected query. Policies/bindings are snapshotted. Builders
// are request-local and must not be shared concurrently.
func (b *PgSelect) SetAccessPolicy(policy access.Policy, binding PolicyBinding) error {
	b.policy = bindPolicy(policy, access.Read, binding)
	return b.policy.err
}

// SetRequestPredicate adds a grouped caller predicate alongside legacy filters.
// It never replaces the independently installed server policy.
func (b *PgSelect) SetRequestPredicate(predicate access.Predicate, fields FieldMap) error {
	b.request = bindPredicate(predicate, fields)
	return b.request.err
}

// SetAccessPolicy attaches a resolved read policy. Even a failed call is
// remembered: BuildSQL returns the error and SQL panics, rather than silently
// generating an unprotected query. Policies/bindings are snapshotted. Builders
// are request-local and must not be shared concurrently.
func (b *PgDetailSelect) SetAccessPolicy(policy access.Policy, binding PolicyBinding) error {
	b.policy = bindPolicy(policy, access.Read, binding)
	return b.policy.err
}

// SetRequestPredicate adds a grouped caller predicate alongside legacy filters.
// It never replaces the independently installed server policy.
func (b *PgDetailSelect) SetRequestPredicate(predicate access.Predicate, fields FieldMap) error {
	b.request = bindPredicate(predicate, fields)
	return b.request.err
}

// SetAccessPolicy attaches a resolved update policy. Even a failed call is
// remembered: BuildSQL returns the error and SQL panics, rather than silently
// generating an unprotected query. Policies/bindings are snapshotted. Builders
// are request-local and must not be shared concurrently.
func (b *PgUpdate) SetAccessPolicy(policy access.Policy, binding PolicyBinding) error {
	b.policy = bindPolicy(policy, access.Update, binding)
	return b.policy.err
}

// SetRequestPredicate adds a grouped caller predicate alongside legacy filters.
// It never replaces the independently installed server policy.
func (b *PgUpdate) SetRequestPredicate(predicate access.Predicate, fields FieldMap) error {
	b.request = bindPredicate(predicate, fields)
	return b.request.err
}

// SetAccessPolicy attaches a resolved delete policy. Even a failed call is
// remembered: BuildSQL returns the error and SQL panics, rather than silently
// generating an unprotected query. Policies/bindings are snapshotted. Builders
// are request-local and must not be shared concurrently.
func (b *PgDelete) SetAccessPolicy(policy access.Policy, binding PolicyBinding) error {
	b.policy = bindPolicy(policy, access.Delete, binding)
	return b.policy.err
}

// SetRequestPredicate adds a grouped caller predicate alongside legacy filters.
// It never replaces the independently installed server policy.
func (b *PgDelete) SetRequestPredicate(predicate access.Predicate, fields FieldMap) error {
	b.request = bindPredicate(predicate, fields)
	return b.request.err
}

// SetAccessPolicy attaches a resolved insert policy. Even a failed call is
// remembered: BuildSQL returns the error and SQL panics, rather than silently
// generating an unprotected query. Policies/bindings are snapshotted. Builders
// are request-local and must not be shared concurrently.
func (b *PgInsert) SetAccessPolicy(policy access.Policy, binding PolicyBinding) error {
	b.policy = bindPolicy(policy, access.Insert, binding)
	return b.policy.err
}

// AccessWriteRules supports policy-aware modelbind preparation. A missing
// policy is an error here, not an implicit unrestricted decision.
func (b PgInsert) AccessWriteRules() (access.Operation, access.WriteRules, error) {
	return installedWriteRules(b.policy, access.Insert)
}

func (b PgUpdate) AccessWriteRules() (access.Operation, access.WriteRules, error) {
	return installedWriteRules(b.policy, access.Update)
}

func installedWriteRules(policy *boundPolicy, operation access.Operation) (access.Operation, access.WriteRules, error) {
	if policy == nil {
		return operation, nil, access.ErrInvalidPolicy
	}
	if err := policy.check(); err != nil {
		return operation, nil, err
	}
	return operation, policy.policy.Writes.Clone(), nil
}
