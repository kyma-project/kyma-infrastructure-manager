package skrdetails

import "testing"

func TestIsHighAvailability(t *testing.T) {
	tests := []struct {
		name  string
		zones []string
		want  bool
	}{
		{"no zones", []string{}, false},
		{"one zone", []string{"a"}, false},
		{"two zones", []string{"a", "b"}, false},
		{"three zones", []string{"a", "b", "c"}, true},
		{"four zones", []string{"a", "b", "c", "d"}, false},
		{"five zones", []string{"a", "b", "c", "d", "e"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHighAvailability(tt.zones); got != tt.want {
				t.Errorf("IsHighAvailability(%v) = %v, want %v", tt.zones, got, tt.want)
			}
		})
	}
}