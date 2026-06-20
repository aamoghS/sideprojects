package math

import "testing"

func TestCeil(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{
		{3.0, 3},
		{3.1, 4},
		{0.01, 1},
		{0.0, 0},
	}
	for _, tc := range cases {
		if got := Ceil(tc.in); got != tc.want {
			t.Fatalf("Ceil(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
