goose postgres "postgres://admin:pwd@localhost:5434/chirpy?sslmode=disable" down
goose postgres "postgres://admin:pwd@localhost:5434/chirpy?sslmode=disable" up
curl -X POST http://localhost:8080/api/users -H "Content-Type: application/json" -d '{ "email": "adi@gmail.com"}'
