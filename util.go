package main

import (
	"strconv"

	"github.com/pyke369/golang-support/ustr"
)

func utilSize(size int64) string {
	switch {
	case size < (1 << 10):
		return strconv.FormatInt(size, 10) + "B"

	case size < (1 << 20):
		return strconv.FormatFloat(float64(size)/(1<<10), 'f', 2, 64) + "kiB"

	case size < (1 << 30):
		return strconv.FormatFloat(float64(size)/(1<<20), 'f', 2, 64) + "MiB"

	default:
		return strconv.FormatFloat(float64(size)/(1<<30), 'f', 1, 64) + "GiB"
	}
}

func utilDuration(duration int) string {
	if duration < 0 {
		return "-:--:--"
	}
	hours := duration / 3600
	duration -= (hours * 3600)
	minutes := duration / 60
	duration -= (minutes * 60)

	return strconv.Itoa(hours) + ":" + ustr.Int(minutes, 2, 1) + ":" + ustr.Int(duration, 2, 1)
}

func utilBandwidth(bandwidth float64) string {
	switch {
	case bandwidth < 1000:
		return strconv.FormatFloat(bandwidth, 'f', 0, 64) + "b/s"

	case bandwidth < (1000 * 1000):
		return strconv.FormatFloat(bandwidth/1000, 'f', 0, 64) + "kb/s"

	case bandwidth < (1000 * 1000 * 1000):
		return strconv.FormatFloat(bandwidth/(1000*1000), 'f', 1, 64) + "Mb/s"

	default:
		return strconv.FormatFloat(bandwidth/(1000*1000*1000), 'f', 1, 64) + "Gb/s"
	}
}
