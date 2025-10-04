package utils

import "strconv"

func DefaultToInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}

	return value
}

func DefaultToInt32(s string, defaultValue int32) int32 {
	if s == "" {
		return defaultValue
	}

	value, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return defaultValue
	}

	return int32(value)
}
