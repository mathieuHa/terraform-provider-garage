package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIErrorIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing bearer token, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NoSuchBucket","message":"Bucket not found: x"}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL+"/", "secret", "test", nil).GetBucket(context.Background(), "x")
	if !IsNotFound(err) {
		t.Fatalf("expected a not found error, got %v", err)
	}
	if got := err.Error(); got != "garage GetBucketInfo: NoSuchBucket (HTTP 404): Bucket not found: x" {
		t.Fatalf("unexpected message %q", got)
	}
}

func TestAPIErrorWithoutJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Forbidden"))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "t", "", nil).ListKeys(context.Background())
	if err == nil || IsNotFound(err) || err.Error() != "garage ListKeys: HTTP 403: Forbidden" {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestZoneRedundancyJSON(t *testing.T) {
	two := int64(2)
	for _, tc := range []struct {
		value ZoneRedundancy
		json  string
	}{
		{ZoneRedundancy{}, `"maximum"`},
		{ZoneRedundancy{AtLeast: &two}, `{"atLeast":2}`},
	} {
		data, err := json.Marshal(tc.value)
		if err != nil || string(data) != tc.json {
			t.Fatalf("marshal: got %s, %v; want %s", data, err, tc.json)
		}
		var back ZoneRedundancy
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatal(err)
		}
		if (back.AtLeast == nil) != (tc.value.AtLeast == nil) || (back.AtLeast != nil && *back.AtLeast != *tc.value.AtLeast) {
			t.Fatalf("round trip of %s changed the value", tc.json)
		}
	}
	if err := json.Unmarshal([]byte(`"minimum"`), &ZoneRedundancy{}); err == nil {
		t.Fatal("expected an error for an unknown value")
	}
}

func TestRoleChangeJSON(t *testing.T) {
	capacity := int64(1000)
	for _, tc := range []struct {
		change RoleChange
		json   string
	}{
		{RoleChange{ID: "n1", Remove: true}, `{"id":"n1","remove":true}`},
		{RoleChange{ID: "n1", Zone: "z", Capacity: &capacity, Tags: []string{"a"}}, `{"capacity":1000,"id":"n1","tags":["a"],"zone":"z"}`},
		{RoleChange{ID: "n1", Zone: "z"}, `{"capacity":null,"id":"n1","tags":[],"zone":"z"}`},
	} {
		data, err := json.Marshal(tc.change)
		if err != nil || string(data) != tc.json {
			t.Fatalf("got %s, %v; want %s", data, err, tc.json)
		}
	}
}

func TestSetBucketKeyPermissions(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body bucketKeyPermRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		data, _ := json.Marshal(body.Permissions)
		calls = append(calls, r.URL.Path+" "+string(data))
		_, _ = w.Write([]byte(`{"id":"b"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "t", "", nil)
	if _, err := c.SetBucketKeyPermissions(context.Background(), "b", "k", BucketPermissions{Read: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SetBucketKeyPermissions(context.Background(), "b", "k", BucketPermissions{}); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`/v2/DenyBucketKey {"read":false,"write":true,"owner":true}`,
		`/v2/AllowBucketKey {"read":true,"write":false,"owner":false}`,
		`/v2/DenyBucketKey {"read":true,"write":true,"owner":true}`,
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v", calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("call %d = %s, want %s", i, calls[i], want[i])
		}
	}
}
