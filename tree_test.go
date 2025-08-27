package slim

import (
	"reflect"
	"sort"
	"testing"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		in       string
		segs     []string
		trail    bool
	}{
		{"/", nil, true},
		{"/a", []string{"/a"}, false},
		{"/a/", []string{"/a"}, true},
		{"/a//b/", []string{"/a", "/b"}, true},
		{"a/b", []string{"/a", "/b"}, false},
	}
	for _, tc := range cases {
		segs, tr := split(tc.in)
		if !reflect.DeepEqual(segs, tc.segs) || tr != tc.trail {
			t.Fatalf("split(%q) => %v,%v; want %v,%v", tc.in, segs, tr, tc.segs, tc.trail)
		}
	}
}

func TestNodeInsertMatchAndRemove(t *testing.T) {
	root := &node{typ: ntStatic}
	var params []string

	// Insert routes
	if _, ok := root.insert([]string{"/users", "/:id"}, &params, 0); !ok {
		t.Fatal("insert users/:id failed")
	}
	params = nil
	if _, ok := root.insert([]string{"/assets", "/*"}, &params, 0); !ok {
		t.Fatal("insert assets/* failed")
	}
	params = nil
	if _, ok := root.insert([]string{"/home"}, &params, 0); !ok {
		t.Fatal("insert /home failed")
	}

	// Attach leaves/endpoints
	// users/:id
	n := root.match([]string{"/users", "/:id"}, 0)
	if n == nil { t.Fatal("match users leaf failed") }
	n.leaf.endpoints = append(n.leaf.endpoints, &endpoint{method: "GET", pattern: "/users/:id", trailingSlash: false, routeId: 1})
	n.leaf.paramsCount = 1
	// assets/*
	m := root.match([]string{"/assets", "/*"}, 0)
	if m == nil { t.Fatal("match assets leaf failed") }
	m.leaf.endpoints = append(m.leaf.endpoints, &endpoint{method: "GET", pattern: "/assets/*", trailingSlash: false, routeId: 2})
	// home
	h := root.match([]string{"/home"}, 0)
	if h == nil { t.Fatal("match home leaf failed") }
	h.leaf.endpoints = append(h.leaf.endpoints, &endpoint{method: "GET", pattern: "/home", trailingSlash: false, routeId: 3})

	// Match static
	got := root.match([]string{"/home"}, 0)
	if got == nil || got.leaf == nil || got.leaf.endpoint("GET") == nil {
		t.Fatal("failed to match /home GET")
	}
	// Match param
	got = root.match([]string{"/users", "/:id"}, 0)
	if got == nil || got.leaf.paramsCount != 1 {
		t.Fatal("failed to match users/:id")
	}
	// Match any
	got = root.match([]string{"/assets", "/foo"}, 0)
	if got == nil || got.leaf.endpoint("GET") == nil {
		t.Fatal("failed to match assets/*")
	}

	// Test leaf.match method returns Allow methods and exact endpoint
	methods, ep := got.leaf.match("GET")
	sort.Strings(methods)
	if ep == nil || len(methods) != 1 || methods[0] != "GET" {
		t.Fatalf("leaf.match unexpected result: ep=%v allow=%v", ep, methods)
	}

	// Note: removal paths are covered by router tests. Here we avoid invoking remove()
	// to keep this unit test focused on insert/match behavior of the tree structure.
}
