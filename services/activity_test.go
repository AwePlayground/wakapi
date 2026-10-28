package services

import "testing"

func TestActivityFillColorDistinguishesEmptyAndActiveDays(t *testing.T) {
	for _, dark := range []bool{true, false} {
		empty := activityFillColor(0, 3600, dark)
		low := activityFillColor(60, 3600, dark)
		high := activityFillColor(3600, 3600, dark)
		if empty == low || low == high {
			t.Fatalf("dark=%v: empty, low and high activity must be visually distinct", dark)
		}
		if activityFillColor(3600, 3600, dark) != high {
			t.Fatal("color changed for same duration")
		}
	}
}
