package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	ServeMux := &http.Server{
		Addr: ":8080",
	}
	http.ListenAndServe(ServeMux.Addr, mux)
}
