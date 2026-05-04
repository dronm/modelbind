# modelbind

`modelbind` binds Go models to common HTTP and PostgreSQL CRUD flows.

It uses struct metadata tags to:

- decode incoming JSON, query, and form values into typed models;
- track fields that were absent in the request;
- distinguish absent fields from explicit `null` values;
- validate text, number, boolean, date, and datetime fields;
- bind validated models to PostgreSQL insert, update, select, filter, sort, and limit builders.

## Installation

```bash
go get github.com/dronm/modelbind
```

The module currently targets Go `1.25+`.

## Basic model

Model fields are discovered by the tag configured in `metadata.FieldAnnotationName`. The default tag is `json`.

```go
package app

import "time"

type UserInput struct {
	ID        *int       `json:"id" primaryKey:"" srvCalc:""`
	Name      *string    `json:"name" alias:"Name" required:"" min:"2" max:"100"`
	Age       *int       `json:"age" alias:"Age" min:"18"`
	Active    *bool      `json:"active"`
	BirthDate *time.Time `json:"birth_date" dateType:"date"`
	CreatedAt *time.Time `json:"created_at" dateType:"datetime_tz" srvCalc:""`
}

func (m *UserInput) Relation() string {
	return "users"
}
```

Common tags:

| Tag | Meaning |
| --- | --- |
| `json:"name"` | Public field/database column name. |
| `alias:"Name"` | Human-readable name for validation messages. |
| `required:""` | Value must be present and non-null. |
| `min:"..."`, `max:"..."`, `fix:"..."` | Text length or numeric constraints. |
| `primaryKey:""` | Marks a primary key field. |
| `srvCalc:""` | Server/database-calculated field. Used as `RETURNING` on insert. |
| `dateType:"date"` | Date/time mode: `date`, `time`, `datetime`, `datetime_tz`. |
| `enum:"status"` | Validate against `metadata.Enums["status"]`. |
| `valList:"a@@b"` | Validate against an inline list. Separator is `metadata.ValListSeparator`. |

## Decode request input

Use `DecodeJSONInput` or `DecodeRequestInput` for HTTP handlers. Despite the name, `DecodeJSONInput` delegates to the generic request decoder:

- `application/json` and `text/json` are decoded from the request body;
- `application/x-www-form-urlencoded` and `multipart/form-data` are decoded from form values;
- requests without a JSON/form body are decoded from URL query values.

```go
package app

import (
	"net/http"

	"github.com/dronm/modelbind"
)

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	input, err := modelbind.DecodeJSONInput[*UserInput](r)
	if err != nil {
		http.Error(w, modelbind.TranslateError(modelbind.LanguageEN, err), http.StatusBadRequest)
		return
	}

	// PATCH {"name": null}
	// Name == nil and the field is present, so it means explicit NULL.
	if input.Model.Name == nil && input.IsPresent("name") {
		// set database column name = NULL
	}

	// PATCH {}
	// Name == nil and the field is absent, so it must be skipped.
	if input.IsAbsent("name") {
		// do not touch database column name
	}
}
```

## Decode URL or form values directly

```go
values := r.URL.Query()

input, err := modelbind.DecodeURLValuesInput[*UserInput](values)
if err != nil {
	return err
}
```

Supported scalar form/query values include strings, booleans, signed/unsigned integers, floats, `time.Time`, pointers to these types, slices, and JSON-encoded structs/maps/interfaces.

Boolean values support Go boolean syntax plus `on`, `off`, `yes`, `no`, `y`, and `n`.

Date formats:

| `dateType` | Accepted input |
| --- | --- |
| `date` | `2006-01-02` |
| `time` | `15:04`, `15:04:05` |
| `datetime` | `2006-01-02 15:04:05`, `2006-01-02T15:04:05`, RFC3339 |
| `datetime_tz` | RFC3339, RFC3339Nano |

For URL/form decoding, the string value `null` means explicit null for pointer, slice, map, and interface fields.

## Validate decoded input

`ModelInput` keeps the decoded model and absent-field metadata together.

```go
input, err := modelbind.DecodeJSONInput[*UserInput](r)
if err != nil {
	return err
}

// true means insert validation: required fields must be present.
if err := input.Validate(true); err != nil {
	return err
}
```

For update validation, use `false`:

```go
if err := input.Validate(false); err != nil {
	return err
}
```

## Insert example

```go
import (
	"github.com/dronm/modelbind"
	"github.com/dronm/modelbind/pg"
)

func BindUserInsert(input modelbind.ModelInput[*UserInput]) (string, []any, error) {
	insert := pg.NewPgInsert(input.Model)

	if err := input.BindInsert(insert); err != nil {
		return "", nil, err
	}

	params := []any{}
	sql := insert.SQL(&params)

	return sql, params, nil
}
```

Example output:

```sql
INSERT INTO users (name,age,active,birth_date) VALUES ($1,$2,$3,$4) RETURNING id,created_at
```

## Update example

For updates, absent fields are skipped. Present fields with explicit `null` are bound as `NULL`.

```go
type UserKey struct {
	ID *int `json:"id"`
}

func (m *UserKey) Relation() string {
	return "users"
}

func BindUserUpdate(id int, input modelbind.ModelInput[*UserInput]) (string, []any, error) {
	key := &UserKey{ID: &id}
	update := pg.NewPgUpdate(input.Model)

	if err := input.BindUpdate(key, update); err != nil {
		return "", nil, err
	}

	params := []any{}
	sql := update.SQL(&params)

	return sql, params, nil
}
```

Example output:

```sql
UPDATE users SET name = $1 WHERE id = $2
```

## Collection select example

`CollectionParams` represents client-side filters, sorters, offset, and limit.

```go
params := modelbind.CollectionParams{
	Sorter: []modelbind.CollectionSorter{
		{Field: "name", Direct: modelbind.SortParAsc},
	},
	From:  0,
	Count: 20,
}

selecter := pg.NewPgSelect(&UserListModel{})
if err := modelbind.BindCollectionSelectModel(selecter, params); err != nil {
	return err
}

queryParams := []any{}
sql := selecter.SQL(&queryParams)
```

## Error translation

Errors are returned as package message errors or validation errors. Use `TranslateError` to render them.

```go
msg := modelbind.TranslateError(modelbind.LanguageEN, err)
msgRU := modelbind.TranslateError(modelbind.LanguageRU, err)
```

## Startup configuration

Optional global configuration should be set during application startup, before concurrent use.

```go
import "github.com/dronm/modelbind/metadata"

func initModelbind() {
	metadata.FieldAnnotationName = "json"
	metadata.ValListSeparator = "@@"
	metadata.Enums = map[string][]string{
		"user_status": {"active", "blocked"},
	}
}
```

## Notes

- Use pointer fields for nullable input values.
- Use `ModelInput.AbsentFields` to distinguish absent fields from explicit `null`.
- Keep insert and update flows separate: inserts usually require all required fields, updates usually skip absent fields.
- SQL builders validate field and relation names before rendering SQL.
