package modelbind

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/dronm/modelbind/pg"
	"github.com/dronm/modelbind/types"
)

// CollectionParamsWithWhere opts into explicit grouped conditions without
// changing CollectionParams or its JSON/unknown-field behavior. Where holds the
// JSON expression, not SQL; absent Where follows the original execution path.
type CollectionParamsWithWhere struct {
	CollectionParams
	Where json.RawMessage `json:"where,omitempty"`
}

// MaxCollectionWhereDepth is the maximum number of group edges from the root.
const MaxCollectionWhereDepth = 64

type collectionWhereNode struct {
	join      string
	children  []collectionWhereNode
	field     string
	condition CollectionFilterField
	path      string
}

// ValidateCollectionWhere checks the client expression structure and operators.
// Model metadata/value validation additionally happens in ParseCollectionWhere.
// Empty bytes mean omitted; explicit JSON null is invalid.
func ValidateCollectionWhere(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	_, err := decodeCollectionWhere(raw, "where", 0)
	return err
}

func decodeCollectionWhere(raw json.RawMessage, path string, depth int) (collectionWhereNode, error) {
	node := collectionWhereNode{path: path}
	fail := func(message string) (collectionWhereNode, error) {
		return node, fmt.Errorf("%s: %s", path, message)
	}
	if depth > MaxCollectionWhereDepth {
		return fail("nesting exceeds 64 levels")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return fail("expected an expression object")
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return fail(err.Error())
		}
		key, ok := token.(string)
		if !ok {
			return fail("expected a member name")
		}
		if _, exists := fields[key]; exists {
			return fail("duplicate member " + key)
		}
		switch key {
		case "and", "or", "f", "o", "v":
		default:
			return fail("unknown member " + key)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return fail(err.Error())
		}
		fields[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return fail(err.Error())
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fail("unexpected content after expression")
	}
	for _, join := range []string{"and", "or"} {
		if rawChildren, ok := fields[join]; ok {
			if len(fields) != 1 {
				return fail("a group must contain only and or only or")
			}
			var children []json.RawMessage
			if err := json.Unmarshal(rawChildren, &children); err != nil || len(children) == 0 {
				return fail(join + " must be a non-empty array")
			}
			node.join = join
			for i, child := range children {
				parsed, err := decodeCollectionWhere(child, fmt.Sprintf("%s.%s[%d]", path, join, i), depth+1)
				if err != nil {
					return node, err
				}
				node.children = append(node.children, parsed)
			}
			return node, nil
		}
	}
	if len(fields) != 3 || fields["f"] == nil || fields["o"] == nil || fields["v"] == nil {
		return fail("a condition requires f, o and v (use null explicitly)")
	}
	if err := json.Unmarshal(fields["f"], &node.field); err != nil || node.field == "" {
		return fail("f must be a non-empty field name")
	}
	if err := json.Unmarshal(fields["o"], &node.condition.Operator); err != nil {
		return fail("o must be a collection operator")
	}
	switch node.condition.Operator {
	case FilterOperParEq, FilterOperParLess, FilterOperParGr, FilterOperParLessEq,
		FilterOperParGrEq, FilterOperParLk, FilterOperParILk, FilterOperParNotEq,
		FilterOperParIs, FilterOperParIn, FilterOperParIncl, FilterOperParAny,
		FilterOperParHas, FilterOperParOverlap, FilterOperParContains, FilterOperParTS:
	default:
		return fail(fmt.Sprintf("unsupported collection operator %q", node.condition.Operator))
	}
	if err := json.Unmarshal(fields["v"], &node.condition.Value); err != nil {
		return fail("invalid value: " + err.Error())
	}
	return node, nil
}

// ParseCollectionWhere validates every leaf with the legacy metadata/value and
// SQL-field validators, then retains its PgFilter behavior in a grouped tree.
// No client-provided SQL expressions or field mappings are accepted.
func ParseCollectionWhere(model types.DBModel, raw json.RawMessage, table string) (pg.FilterExpression, error) {
	node, err := decodeCollectionWhere(raw, "where", 0)
	if err != nil {
		return pg.FilterExpression{}, err
	}
	if table != "" {
		if _, err := pg.SanitizeSQLFieldRef(table + ".field"); err != nil {
			return pg.FilterExpression{}, fmt.Errorf("where: invalid table: %w", err)
		}
	}
	var compile func(collectionWhereNode) (pg.FilterExpression, error)
	compile = func(node collectionWhereNode) (pg.FilterExpression, error) {
		if node.join != "" {
			children := make([]pg.FilterExpression, len(node.children))
			for i, child := range node.children {
				parsed, err := compile(child)
				if err != nil {
					return pg.FilterExpression{}, err
				}
				children[i] = parsed
			}
			if node.join == "and" {
				return pg.FilterAnd(children...), nil
			}
			return pg.FilterOr(children...), nil
		}
		filters := pg.PgFilters{}
		params := CollectionParams{Filter: []CollectionFilter{{Fields: map[string]CollectionFilterField{node.field: node.condition}}}}
		if err := ParseFilterParams(model, &filters, params, table); err != nil {
			return pg.FilterExpression{}, fmt.Errorf("%s: %w", node.path, err)
		}
		return pg.FilterLeaf(filters), nil
	}
	return compile(node)
}

// BindCollectionSelectModelWithWhere is the PostgreSQL opt-in counterpart of
// BindCollectionSelectModel. All existing binding logic stays in the old API.
func BindCollectionSelectModelWithWhere(dbSelect *pg.PgGroupedSelect, params CollectionParamsWithWhere) error {
	if dbSelect == nil || dbSelect.PgSelect == nil {
		return fmt.Errorf("grouped select is not initialized")
	}
	if len(params.Where) != 0 {
		// Fail closed if parsing fails and a caller accidentally ignores the error.
		_ = dbSelect.SetWhere(pg.FilterExpression{})
		expression, err := ParseCollectionWhere(dbSelect.Model(), params.Where, "")
		if err != nil {
			return err
		}
		if err := dbSelect.SetWhere(expression); err != nil {
			return err
		}
	}
	return BindCollectionSelectModel(dbSelect, params.CollectionParams)
}
