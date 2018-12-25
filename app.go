package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// TrendingSongsEndPoint gets the Trending songs
func TrendingSongsEndPoint(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Not implemented yet!")
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/trending/{period}", TrendingSongsEndPoint).Methods("GET")
	if err := http.ListenAndServe(":3001", r); err != nil {
		log.Fatal(err)
	}
}
