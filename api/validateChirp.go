package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

const (
	maxLen = 140
)

type ResponseStruct struct {
	Body  any    `json:"body,omitempty"`
	Error string `json:"error,omitempty"`
	Valid bool   `json:"valid"`
}

func RespondWithError(w http.ResponseWriter, err string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	res := ResponseStruct{
		Error: err,
		Valid: false,
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&res)
}

func RespondWithJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resStruct := ResponseStruct{
		Body:  data,
		Valid: true,
	}

	resp, err := json.MarshalIndent(resStruct, "", "  ")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

var profaneList = []*regexp.Regexp{
	regexp.MustCompile(`(?i)kerfuffle`),
	regexp.MustCompile(`(?i)sharbert`),
	regexp.MustCompile(`(?i)fornax`),
}
var profaneListString = []string{"kerfuffle", "sharbert", "fornax"}

func CleanseChirp(chirp string) string {
	for _, re := range profaneListString {
		chirp = strings.ReplaceAll(chirp, re, "****")
	}
	return chirp

}
