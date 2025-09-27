package client

import "testing"

func TestBuildURLPreservesQuery(t *testing.T) {
	base, err := newBase("http://localhost:8080")
	if err != nil {
		t.Fatalf("newBase: %v", err)
	}

	got := base.buildURL("/admin/tenants/foo/keys?app_id=bar")
	want := "http://localhost:8080/admin/tenants/foo/keys?app_id=bar"
	if got != want {
		t.Fatalf("buildURL mismatch:\nwant %s\n got %s", want, got)
	}
}

func TestBuildURLWithBasePath(t *testing.T) {
	base, err := newBase("http://localhost:8080/api")
	if err != nil {
		t.Fatalf("newBase: %v", err)
	}

	got := base.buildURL("collections?limit=10")
	want := "http://localhost:8080/api/collections?limit=10"
	if got != want {
		t.Fatalf("buildURL mismatch:\nwant %s\n got %s", want, got)
	}
}
