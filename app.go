package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	. "github.com/lttkgp/R2-D2/config"
	. "github.com/lttkgp/R2-D2/dao"
)

var config = Config{}
var dao = TrendingDao{}

// TrendingSongsEndPoint gets the Trending songs
func TrendingSongsEndPoint(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	posts, err := dao.GetTrendingForPeriod(params["period"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Song ID")
		return
	}
	respondWithJSON(w, http.StatusOK, posts)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, map[string]string{"error": msg})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// Parse the configuration file 'config.toml', and establish a connection to DB
func init() {
	config.Read()

	dao.Server = config.Server
	dao.Database = config.Database
	dao.Connect()
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/trending/{period}", TrendingSongsEndPoint).Methods("GET")
	if err := http.ListenAndServe(":3001", r); err != nil {
		log.Fatal(err)
	}
}
