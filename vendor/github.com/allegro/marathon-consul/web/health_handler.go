package web

import (
	"fmt"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "OK")
}
