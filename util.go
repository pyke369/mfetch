package main

import "fmt"

func utilSize(size int64) string {
	if size < (1 << 10) {
		return fmt.Sprintf("%dB", size)
	} else if size < (1 << 20) {
		return fmt.Sprintf("%.2fkiB", float64(size)/(1<<10))
	} else if size < (1 << 30) {
		return fmt.Sprintf("%.2fMiB", float64(size)/(1<<20))
	} else {
		return fmt.Sprintf("%.1fGiB", float64(size)/(1<<30))
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
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, duration)
}

func utilBandwidth(bandwidth float64) string {
	if bandwidth < 1000 {
		return fmt.Sprintf("%.0fb/s", bandwidth)
	} else if bandwidth < (1000 * 1000) {
		return fmt.Sprintf("%.0fkb/s", bandwidth/(1000))
	} else if bandwidth < (1000 * 1000 * 1000) {
		return fmt.Sprintf("%.1fMb/s", bandwidth/(1000*1000))
	} else {
		return fmt.Sprintf("%.1fGb/s", bandwidth/(1000*1000*1000))
	}
}
