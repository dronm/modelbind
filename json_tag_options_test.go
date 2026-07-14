package modelbind

import (
	"strings"
	"testing"

	"github.com/dronm/modelbind/pg"
)

type jsonTagOptionsAgg struct {
	Total int `json:"total,omitempty" agg:"count(*)"`
}

type jsonTagOptionsModel struct {
	ID        *int    `json:"id,omitempty" primaryKey:""`
	Name      *string `json:"name,omitempty" required:""`
	ServerID  *int    `json:"server_id,omitempty" srvCalc:""`
	Ignored   *string `json:"-,omitempty"`
	EmptyName *string `json:",omitempty"`
}

func (m *jsonTagOptionsModel) Relation() string {
	return "json_tag_options"
}

func (m *jsonTagOptionsModel) CollectionAgg() any {
	return &jsonTagOptionsAgg{}
}

type jsonTagOptionsKey struct {
	ID *int `json:"id,omitempty"`
}

func TestJSONTagOptionsAreRemovedFromBoundFieldNames(t *testing.T) {
	id := 7
	name := "example"

	t.Run("insert", func(t *testing.T) {
		model := &jsonTagOptionsModel{Name: &name}
		dbInsert := pg.NewPgInsert(model)

		if err := BindInsertModel(dbInsert); err != nil {
			t.Fatalf("BindInsertModel() error = %v", err)
		}

		params := []any{}
		sql := dbInsert.SQL(&params)
		want := "INSERT INTO json_tag_options (name) VALUES ($1) RETURNING server_id"
		if sql != want {
			t.Fatalf("SQL = %q, want %q", sql, want)
		}
		if strings.Contains(sql, "omitempty") {
			t.Fatalf("SQL contains JSON tag option: %q", sql)
		}
	})

	t.Run("update", func(t *testing.T) {
		model := &jsonTagOptionsModel{Name: &name}
		dbUpdate := pg.NewPgUpdate(model)

		if err := BindUpdateModel(&jsonTagOptionsKey{ID: &id}, dbUpdate); err != nil {
			t.Fatalf("BindUpdateModel() error = %v", err)
		}

		params := []any{}
		sql := dbUpdate.SQL(&params)
		want := "UPDATE json_tag_options SET name = $1 WHERE id = $2"
		if sql != want {
			t.Fatalf("SQL = %q, want %q", sql, want)
		}
		if strings.Contains(sql, "omitempty") {
			t.Fatalf("SQL contains JSON tag option: %q", sql)
		}
	})

	t.Run("detail select", func(t *testing.T) {
		model := &jsonTagOptionsModel{}
		dbSelect := pg.NewPgDetailSelect(model, &pg.PgFilters{})

		if err := BindDetailSelectModel(&jsonTagOptionsKey{ID: &id}, dbSelect); err != nil {
			t.Fatalf("BindDetailSelectModel() error = %v", err)
		}

		params := []any{}
		sql := dbSelect.SQL(&params)
		want := "SELECT id,name,server_id FROM json_tag_options WHERE id = $1"
		if sql != want {
			t.Fatalf("SQL = %q, want %q", sql, want)
		}
		if strings.Contains(sql, "omitempty") {
			t.Fatalf("SQL contains JSON tag option: %q", sql)
		}
	})

	t.Run("collection select and aggregation", func(t *testing.T) {
		model := &jsonTagOptionsModel{}
		dbSelect := pg.NewPgSelect(
			model,
			&pg.PgFilters{},
			&pg.PgSorters{},
			pg.NewPgLimit(0, 0),
		)

		if err := BindCollectionSelectModel(dbSelect, CollectionParams{}); err != nil {
			t.Fatalf("BindCollectionSelectModel() error = %v", err)
		}

		params := []any{}
		selectSQL, aggSQL := dbSelect.CollectionSQL(&params)
		wantSelect := "SELECT id,name,server_id FROM json_tag_options"
		if selectSQL != wantSelect {
			t.Fatalf("select SQL = %q, want %q", selectSQL, wantSelect)
		}

		wantAgg := "SELECT count(*) FROM json_tag_options"
		if aggSQL != wantAgg {
			t.Fatalf("aggregation SQL = %q, want %q", aggSQL, wantAgg)
		}

		if strings.Contains(selectSQL+aggSQL, "omitempty") {
			t.Fatalf("SQL contains JSON tag option: select=%q aggregation=%q", selectSQL, aggSQL)
		}
	})
}
