package main

import "strconv"

func BytesToReadable(bytes int) string {
	if bytes < 1024 {
		return strconv.Itoa(bytes) + " B"
	} else if bytes < 1024*1024 {
		return strconv.FormatFloat(float64(bytes)/1024, 'f', 2, 64) + " KB"
	} else if bytes < 1024*1024*1024 {
		return strconv.FormatFloat(float64(bytes)/(1024*1024), 'f', 2, 64) + " MB"
	} else {
		return strconv.FormatFloat(float64(bytes)/(1024*1024*1024), 'f', 2, 64) + " GB"
	}
}
