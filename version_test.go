package flourish

import "testing"

func TestVersion_IsSemver(t *testing.T) {
	if !VersionIsSemver() {
		t.Fatalf("embedded version must be semver: got %q", Version())
	}
}

func TestVersionTag_PrefixesV(t *testing.T) {
	if got, want := VersionTag(), "v"+Version(); got != want {
		t.Fatalf("version tag: got %q, want %q", got, want)
	}
}

func TestIsSemver(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{version: "0.1.0", want: true},
		{version: "1.2.3-alpha.1", want: true},
		{version: "2.0.0+build.7", want: true},
		{version: "v1.2.3", want: false},
		{version: "1.2", want: false},
		{version: "01.2.3", want: false},
	}

	for _, tc := range cases {
		got := IsSemver(tc.version)
		if got != tc.want {
			t.Fatalf("IsSemver(%q): got %v, want %v", tc.version, got, tc.want)
		}
	}
}
