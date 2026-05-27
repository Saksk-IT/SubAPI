package service

import "testing"

func TestPsComputeValidityDaysSupportsSingularAndPluralUnits(t *testing.T) {
	tests := []struct {
		name string
		days int
		unit string
		want int
	}{
		{name: "days", days: 30, unit: "days", want: 30},
		{name: "week", days: 2, unit: "week", want: 14},
		{name: "weeks", days: 2, unit: "weeks", want: 14},
		{name: "month", days: 3, unit: "month", want: 90},
		{name: "months", days: 3, unit: "months", want: 90},
		{name: "year", days: 1, unit: "year", want: 365},
		{name: "years", days: 2, unit: "years", want: 730},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := psComputeValidityDays(tt.days, tt.unit); got != tt.want {
				t.Fatalf("psComputeValidityDays(%d, %q) = %d, want %d", tt.days, tt.unit, got, tt.want)
			}
		})
	}
}
