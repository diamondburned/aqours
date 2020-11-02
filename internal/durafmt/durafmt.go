package durafmt

import (
	"fmt"
	"strings"
	"time"
)

var durationChunks = []time.Duration{time.Hour, time.Minute, time.Second}

// Format formats the given duration into HH:MM:SS form.
func Format(d time.Duration) string {
	var dwords = make([]string, 0, 3)
	var n int

	for i, section := range durationChunks {
		n, d = divide(d, section)
		// Skip hour if there's none.
		if i == 0 && n < 1 {
			continue
		}

		dwords = append(dwords, fmt.Sprintf("%02d", n))
	}

	return strings.Join(dwords, ":")
}

func divide(d, div time.Duration) (n int, newd time.Duration) {
	n = int(d / div)
	return n, d - time.Duration(n)*div
}
