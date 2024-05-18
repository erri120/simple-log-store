package api

import "net/http"

func writeInternalServerError(w http.ResponseWriter) {
	http.Error(w, "something went wrong", http.StatusInternalServerError)
}
