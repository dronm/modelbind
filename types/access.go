package types

import "github.com/dronm/modelbind/access"

// AccessScopedModel optionally associates multiple model representations (write,
// list, detail and key models) with one application-level policy resource.
// Existing DBModel implementations need not implement this interface. Merely
// implementing it does not install a policy: application/webapp integration must
// resolve the resource and attach a policy before executing a protected query.
type AccessScopedModel interface {
	AccessResource() string
}

// AccessWritePolicy is an optional capability for policy-aware model binding.
// It returns a snapshot of the installed rules or an error when no valid policy
// is installed. Implementations must ALSO enforce these rules when rendering SQL.
type AccessWritePolicy interface {
	AccessWriteRules() (access.Operation, access.WriteRules, error)
}
