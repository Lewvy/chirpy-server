package main

import (
	"database/sql"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync/atomic"

	"github.com/Lewvy/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/valkey-io/valkey-go"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	cache          valkey.Client
}

func main() {
	godotenv.Load(".env")
	mux := http.NewServeMux()
	serveMux := &http.Server{
		Addr: ":8080",
	}

	filePathRoot := "."

	dbUrl := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Error connecting to the database: %q", err.Error())
	}
	defer db.Close()
	log.Println("Database initialized successfully")

	valkeyClient, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{os.Getenv("CACHE_ADDR")}})

	if err != nil {
		log.Fatalf("Error initializing cache")
	}
	log.Println("Cache initialized successfully")

	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      database.New(db),
		cache:          valkeyClient,
	}
	defer valkeyClient.Close()
	go cfg.Worker()

	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filePathRoot)))

	mux.Handle("/app/", cfg.middlewareMetricsInc(handler))
	mux.HandleFunc("/app/assets", GetAssets)

	mux.HandleFunc("POST /api/chirps", cfg.PostChirps)

	mux.HandleFunc("POST /api/users/register", cfg.RegisterUser)
	mux.HandleFunc("POST /api/users/login", cfg.Login)
	mux.HandleFunc("PATCH /api/users/password-reset", cfg.PasswordReset)

	mux.HandleFunc("GET /api/chirps", cfg.GetAllChirps)
	mux.HandleFunc("GET /api/chirps/{id}", cfg.GetChirp)

	mux.HandleFunc("GET /api/healthz", Readiness)

	mux.HandleFunc("GET /admin/metrics", cfg.Metrics)
	mux.Handle("POST /admin/reset", cfg.middlewareCheckPlatform(cfg.DeleteAllUsers()))

	mux.Handle("/debug/pprof/", http.DefaultServeMux)

	log.Printf("Serving files from %s on port %s\n", filePathRoot, serveMux.Addr)
	log.Fatal(http.ListenAndServe(serveMux.Addr, mux))
}
