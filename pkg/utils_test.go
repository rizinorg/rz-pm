package pkg

import "testing"

func TestGetMajorMinorVersion(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"1.2.3", "1.2"},
		{"10.20.30", "10.20"},
		{"0.1.2", "0.1"},
		{"5.6", "5.6"}, // Edge case with no patch version
		{"1", "1.0"},   // Edge case with only major version
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := GetMajorMinorVersion(tt.version)
			if got != tt.want {
				t.Errorf("GetMajorMinorVersion(%q) = %q; want %q", tt.version, got, tt.want)
			}
		})
	}
}
