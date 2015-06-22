package utils

import (
	"strings"
)

func CleanID(ID string) string {
	return strings.Replace(
		strings.Trim(ID, "/"),
		"/", "-", -1,
	)
}
