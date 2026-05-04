package pg

import "testing"

func TestSanitizeSQLFieldRefMoreCases(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		want    string
		wantErr bool
	}{
		{name: "trims spaces", field: " field_name ", want: "field_name"},
		{name: "json nested", field: "payload->'items'->0->>'name'", want: "payload->'items'->0->>'name'"},
		{name: "json key with colon", field: "payload->>'ns:key'", want: "payload->>'ns:key'"},
		{name: "empty", field: "", wantErr: true},
		{name: "double dot", field: "users..id", wantErr: true},
		{name: "quoted identifier rejected", field: `"users"."id"`, wantErr: true},
		{name: "json invalid key", field: "payload->>'bad key'", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := SanitizeSQLFieldRef(test.field)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestJoinSafeFieldRefs(t *testing.T) {
	got := joinSafeFieldRefs([]string{"id", "u.name", "payload->>'name'"})
	want := "id,u.name,payload->>'name'"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestJoinSafeFieldRefsPanicsOnUnsafeField(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unsafe field reference")
		}
	}()

	_ = joinSafeFieldRefs([]string{"id", "name;DROP TABLE users"})
}
