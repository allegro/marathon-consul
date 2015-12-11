package web

import (
	"fmt"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}
