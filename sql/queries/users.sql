-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
  $1, $2, $3, $4, $5
  )
RETURNING *;

-- name: ClearTable :exec
TRUNCATE TABLE users RESTART IDENTITY CASCADE;

-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at,body,  user_id)
VALUES (
    $1, $2, $3, $4, $5
    )
RETURNING *;

-- name: GetAllChirps :many
Select * from chirps order by created_at;

-- name: GetChirpByID :one
Select * from chirps where id = $1;

-- name: GetUserByEmail :one
Select id, hashed_password, email, created_at, updated_at from users where email = $1;

-- name: UpdateUserPw :exec
Update users
set hashed_password = $1
where email = $2;
