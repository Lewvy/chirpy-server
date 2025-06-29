-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users(
    id uuid primary key default gen_random_uuid(),
    created_at TIMESTAMP not null default NOW(),
    updated_at TIMESTAMP not null default NOW(),
    email varchar(255) not null UNIQUE,
    hashed_password varchar(255) not null
);

-- +goose Down
DROP TABLE users;

