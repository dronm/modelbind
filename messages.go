package modelbind

import "github.com/dronm/modelbind/metadata"

// MessageID is a stable identifier for a message template.
//
// The root package intentionally reuses metadata.MessageID so errors from this
// package and from the metadata subpackage can be translated by the same API.
type MessageID = metadata.MessageID

const (
	// Model / metadata mapping errors.
	MsgMetadataFieldNotFound  MessageID = "quvalid.metadata.field_not_found"
	MsgInvalidFieldExpression MessageID = "quvalid.sql.invalid_field_expression"
	MsgModelFieldNotInterface MessageID = "quvalid.model.field_not_interface"

	// CRUD preparation errors.
	MsgNoInsertFields MessageID = "quvalid.crud.no_insert_fields"
	MsgNoUpdateFields MessageID = "quvalid.crud.no_update_fields"
	MsgNoKeys         MessageID = "quvalid.crud.no_keys"

	// Collection select / aggregation errors.
	MsgAggregationFieldNotDefined    MessageID = "quvalid.collection.aggregation.field_not_defined"
	MsgAggregationFunctionNotDefined MessageID = "quvalid.collection.aggregation.function_not_defined"

	// Builder initialization errors.
	MsgFilterNotInitialized MessageID = "quvalid.builder.filter_not_initialized"
	MsgSorterNotInitialized MessageID = "quvalid.builder.sorter_not_initialized"
	MsgLimitNotInitialized  MessageID = "quvalid.builder.limit_not_initialized"

	// Generic operation errors.
	MsgJSONMarshalFailed MessageID = "quvalid.json.marshal_failed"
)

// DefaultMessageTemplates contains English fallback templates for all root
// package messages.
//
// Common template data keys:
//   - Func: function name
//   - Field: model/db field ID
//   - Expression: SQL field expression
//   - Index: zero-based field index
//   - Error: nested error text
var DefaultMessageTemplates = map[MessageID]string{
	MsgMetadataFieldNotFound:  "{{ .Func }}() failed: field {{ .Field }} not found in metadata",
	MsgInvalidFieldExpression: "{{ .Func }}() failed: invalid field expression {{ .Expression }}",
	MsgModelFieldNotInterface: "reflect.CanInterface() failed for field {{ .Field }}",

	MsgNoInsertFields: "{{ .Func }}() failed: no fields to insert",
	MsgNoUpdateFields: "{{ .Func }}() failed: no fields to update",
	MsgNoKeys:         "{{ .Func }}() failed: keys not found",

	MsgAggregationFieldNotDefined:    "{{ .Func }}() failed: aggregation field not defined for index {{ .Index }}",
	MsgAggregationFunctionNotDefined: "{{ .Func }}() failed: aggregation function not defined for field {{ .Field }}",

	MsgFilterNotInitialized: "{{ .Func }}() failed: filter should be initialized",
	MsgSorterNotInitialized: "{{ .Func }}() failed: sorter should be initialized",
	MsgLimitNotInitialized:  "{{ .Func }}() failed: limit should be initialized",

	MsgJSONMarshalFailed: "{{ .Func }}() failed: field {{ .Field }} can not be marshaled to JSON: {{ .Error }}",
}

// Deprecated legacy fmt-style message constants.
//
// They are kept only to avoid breaking external code that may still reference
// the old constants. New package code should return NewMessageError(...) with
// one of the Msg* IDs above.
const (
	ErrNoFieldInMD        = "%s() failed: field %s not found in metadata"
	ErrInvalidFieldExpr   = "%s() failed: invalid field expression %s"
	ErrNoInsertFields     = "%s() failed: no fields to insert"
	ErrNoUpdateFields     = "%s() failed: no fields to update"
	ErrNoKeys             = "%s() failed: keys not found"
	ErrAggFieldNotDefined = "PrepareFetchModelCollection() failed: aggregation field not defined for index %d"
	ErrAggFieldNoFnc      = "PrepareFetchModelCollection() failed: aggregation function not defined for field: %s"
	ErrFilterNotInit      = "%s() failed: Filter should be initialized"
	ErrSorterNotInit      = "%s() failed: Sorter should be initialized"
	ErrLimitNotInit       = "%s() failed: Limit should be initialized"
)
