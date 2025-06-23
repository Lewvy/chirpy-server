package main

import (
	"html/template"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync/atomic"

	"github.com/Lewvy/chirpy/api"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	mux := http.NewServeMux()
	ServeMux := &http.Server{
		Addr: ":8080",
	}
	filePathRoot := "."
	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filePathRoot)))
	mux.Handle("/app/", cfg.middlewareMetricsInc(handler))
	mux.HandleFunc("/app/assets", func(w http.ResponseWriter, r *http.Request) {
		path := "assets/chirp.html"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Println("file not found: ", path)
			http.NotFound(w, r)
			return
		}
		log.Println("Serving file: ", path)
		http.ServeFile(w, r, path)
	})
	mux.Handle("GET /api/healthz", http.HandlerFunc(Readiness))
	mux.HandleFunc("POST /api/validate_chirp", api.ValidateChirp)
	mux.HandleFunc("GET /admin/metrics", cfg.Metrics)
	mux.HandleFunc("POST /admin/reset", cfg.Reset)
	mux.Handle("/debug/pprof/", http.DefaultServeMux)
	log.Printf("Serving files from %s on port %s\n", filePathRoot, ServeMux.Addr)
	log.Fatal(http.ListenAndServe(ServeMux.Addr, mux))
}

func Readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Fatalf("Unexpected error: %q", err)
	}
}

func (cfg *apiConfig) Metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	tmpl := template.Must(template.New("metrics").Parse(
		`
		<html>
			<body> 
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited {{.Hits}} times!</p> 
			</body>
		</html> 
		`))
	data := struct {
		Hits int64
	}{
		Hits: int64(cfg.fileserverHits.Load()),
	}
	tmpl.Execute(w, data)
}

func (cfg *apiConfig) Reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}
