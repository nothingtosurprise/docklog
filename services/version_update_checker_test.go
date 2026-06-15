package services

import "testing"

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"1.0.0", true},
		{"v1.2.3", true},
		{"latest", false},
		{"1.2", false},
		{"1.2.3-beta", false},
	}
	for _, tc := range cases {
		_, ok := ParseSemver(tc.in)
		if ok != tc.ok {
			t.Fatalf("ParseSemver(%q) ok=%v want=%v", tc.in, ok, tc.ok)
		}
	}
}

func TestIsSemverNewer(t *testing.T) {
	a, _ := ParseSemver("1.3.0")
	b, _ := ParseSemver("1.2.9")
	if !IsSemverNewer(a, b) {
		t.Fatal("expected newer version")
	}
	if IsSemverNewer(b, a) {
		t.Fatal("did not expect older version to be newer")
	}
}
