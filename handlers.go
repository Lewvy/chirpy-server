package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/Lewvy/chirpy/api"
	"github.com/Lewvy/chirpy/internal/auth"
	"github.com/Lewvy/chirpy/internal/database"
	"github.com/google/uuid"
)

type smtpServer struct {
	host string
	port string
}

func (s *smtpServer) Address() string {
	return s.host + ":" + s.port
}

func (cfg *apiConfig) GetChirp(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}
	chirp, err := cfg.dbQueries.GetChirpByID(context.Background(), id)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusNotFound)
		return
	}
	api.RespondWithJSON(w, chirp, http.StatusOK)
}

func (cfg *apiConfig) GetAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.dbQueries.GetAllChirps(context.Background())
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.RespondWithJSON(w, chirps, http.StatusOK)
}

func (cfg *apiConfig) PostChirps(w http.ResponseWriter, r *http.Request) {
	maxLen := 140
	dataStr := struct {
		Body    string    `json:"body"`
		User_id uuid.UUID `json:"user_id"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&dataStr)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	chirpstr := strings.TrimSpace(string(dataStr.Body))
	if len(chirpstr) > maxLen {
		api.RespondWithError(w, "Chirp is too long", http.StatusBadRequest)
		return
	}

	dataStr.Body = api.CleanseChirp(chirpstr)
	chirp := database.CreateChirpParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Body:      dataStr.Body,
		UserID:    dataStr.User_id,
	}
	chirpResp, err := cfg.dbQueries.CreateChirp(context.Background(), chirp)
	if err != nil {
		api.RespondWithError(w, "Unexpected error occured: "+err.Error(), http.StatusInternalServerError)
		return
	}

	api.RespondWithJSON(w, chirpResp, 200)
}

type UserLogins struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func getUserCreds(r io.Reader) (*UserLogins, error) {
	user := &UserLogins{}
	data, err := io.ReadAll(r)
	if err != nil {
		return &UserLogins{}, err
	}
	err = json.Unmarshal(data, &user)
	if err != nil {
		return &UserLogins{}, err
	}
	return user, nil
}

func (cfg *apiConfig) Login(w http.ResponseWriter, r *http.Request) {
	user, err := getUserCreds(r.Body)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if user.Password == "" || user.Email == "" {
		api.RespondWithError(w, "Password & email are required", http.StatusBadRequest)
		return
	}
	userDetails, err := cfg.dbQueries.GetUserByEmail(context.Background(), user.Email)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hashed_pwd := userDetails.HashedPassword
	if hashed_pwd == "unset" {
		cfg.sendOtpForPasswordChange(w, userDetails)
		return
	}
	isValid, err := auth.VerifyHashedPw(hashed_pwd, user.Password)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !isValid {
		api.RespondWithError(w, "Incorrect Password", http.StatusUnauthorized)
		return
	}
	api.RespondWithJSON(w, "Logged in successfully", http.StatusOK)
}

func generateOTP() (string, error) {
	max := 6
	table := []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}
	b := make([]byte, max)
	n, err := io.ReadAtLeast(rand.Reader, b, max)
	if n != max {
		panic(err)
	}
	for i := range b {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b), nil
}

type EmailJob struct {
	Email     string
	EmailReal string
}

var jobQueue = make(chan EmailJob, 100)

func (cfg *apiConfig) Worker() {
	from := os.Getenv("COMPANY_EMAIL")
	password := os.Getenv("COMPANY_PWD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpServer := smtpServer{host: smtpHost, port: smtpPort}
	for job := range jobQueue {
		otp, err := generateOTP()
		if err != nil {
			continue
		}
		msg := fmt.Sprintf("Subject: OTP for password change\r\n\r\n OTP: %s", otp)
		message := []byte(msg)
		auth := smtp.PlainAuth("", from, password, smtpServer.host)
		if err := smtp.SendMail(smtpServer.Address(), auth, from, []string{job.Email}, message); err != nil {
			continue

		}
		err = cfg.saveToCache(otp, job.EmailReal)
		if err != nil {
			log.Println("Error saving to cache: OTP: ", otp)
		} else {
			log.Println("Otp sent successfully: ", otp)
		}

	}
}

func (cfg *apiConfig) sendOtpForPasswordChange(w http.ResponseWriter, user database.GetUserByEmailRow) {
	to := os.Getenv("SAMPLE_EMAIL")
	//TODO: need to change the to to user.Email for production
	jobQueue <- EmailJob{Email: to, EmailReal: user.Email}
	api.RespondWithJSON(w, "Email Sent!", http.StatusCreated)
}

func (cfg *apiConfig) saveToCache(otp, email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := cfg.cache.Do(ctx, cfg.cache.B().Set().Key(email).Value(otp).Build()).Error()
	if err != nil {
		return err
	}
	return nil
}

func (cfg *apiConfig) PasswordReset(w http.ResponseWriter, r *http.Request) {
	userStruct := struct {
		Otp   string `json:"otp"`
		Pwd   string `json:"password"`
		Email string `json:"email"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&userStruct)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	otp, err := cfg.cache.Do(ctx, cfg.cache.B().Get().Key(userStruct.Email).Build()).ToString()
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if otp != userStruct.Otp {
		api.RespondWithError(w, "Wrong otp provided", http.StatusForbidden)
		return
	}
	hashed_pwd, err := auth.HashPassword(userStruct.Pwd)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusForbidden)
		return
	}
	if hashed_pwd == nil {
		api.RespondWithError(w, "Password is required", http.StatusForbidden)
		return
	}
	userResp := database.UpdateUserPwParams{
		HashedPassword: *hashed_pwd,
		Email:          userStruct.Email,
	}
	err = cfg.dbQueries.UpdateUserPw(ctx, userResp)
	if err != nil {
		api.RespondWithError(w, "Error updating password: "+err.Error(), http.StatusInternalServerError)
		return
	}
	api.RespondWithJSON(w, "Successfully updated password", http.StatusOK)
}

func (cfg *apiConfig) RegisterUser(w http.ResponseWriter, r *http.Request) {
	user, err := getUserCreds(r.Body)
	hashed_pwd, err := auth.HashPassword(user.Password)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	DBuser := database.CreateUserParams{
		Email:          user.Email,
		ID:             uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		HashedPassword: *hashed_pwd,
	}

	usr, err := cfg.dbQueries.CreateUser(context.Background(), DBuser)
	if err != nil {
		api.RespondWithError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userResponse := struct {
		Email     string    `json:"email"`
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		Email:     usr.Email,
		ID:        usr.ID,
		CreatedAt: usr.CreatedAt,
		UpdatedAt: usr.UpdatedAt,
	}
	api.RespondWithJSON(w, userResponse, http.StatusCreated)
}

func (cfg *apiConfig) DeleteAllUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := cfg.dbQueries.ClearTable(context.Background()); err != nil {
			api.RespondWithError(w, err.Error(), http.StatusInternalServerError)
		}
		cfg.ResetServerHits()
		api.RespondWithJSON(w, "Server reset successful", http.StatusOK)
	}
}

func GetAssets(w http.ResponseWriter, r *http.Request) {
	path := "assets/chirp.html"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println("file not found: ", path)
		http.NotFound(w, r)
		return
	}

	log.Println("Serving file: ", path)
	http.ServeFile(w, r, path)
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

	tmpl := template.Must(template.New("metrics").Parse(`
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

func (cfg *apiConfig) ResetServerHits() {
	cfg.fileserverHits.Store(0)
}
