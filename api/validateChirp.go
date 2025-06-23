package api

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const (
	maxLen = 140
)

type ResponseStruct struct {
	Body  string `json:"cleaned_body,omitempty"`
	Error string `json:"error,omitempty"`
	Valid bool   `json:"valid"`
}

func respondWithError(w http.ResponseWriter, err string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	res := ResponseStruct{
		Error: err,
		Valid: false,
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&res)
}

func respondWithJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	res := ResponseStruct{
		Body:  data.(string),
		Valid: true,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

var profaneList = []*regexp.Regexp{
	regexp.MustCompile(`(?i)kerfuffle`),
	regexp.MustCompile(`(?i)sharbert`),
	regexp.MustCompile(`(?i)fornax`),
}
var profaneListString = []string{"kerfuffle", "sharbert", "fornax"}

func cleanseChirp(chirp string) string {
	for _, re := range profaneListString {
		chirp = strings.ReplaceAll(chirp, re, "****")
	}
	return chirp

}

func ValidateChirp(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}
	chirp := strings.TrimSpace(string(data))
	if len(chirp) > maxLen {
		respondWithError(w, "Chirp is too long", http.StatusBadRequest)
		return
	}
	cleansedChirp := cleanseChirp(chirp)

	respondWithJSON(w, cleansedChirp)
}
