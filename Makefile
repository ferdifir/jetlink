.PHONY: run build migrate setup-db seed-superadmin

run:
	go run ./cmd/main.go

build:
	go build -o bin/jetlink ./cmd/main.go

migrate:
	mysql -u$$DB_USER -p$$DB_PASSWORD $$DB_NAME < migrations/001_init.sql

setup-db:
	mysql -u$$DB_USER -p$$DB_PASSWORD -e "CREATE DATABASE IF NOT EXISTS jetlink CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
	mysql -u$$DB_USER -p$$DB_PASSWORD jetlink < migrations/001_init.sql

seed-superadmin:
	@read -p "Email: " email; \
	read -s -p "Password: " pass; \
	echo; \
	HASH=$$(go run -e 'package main; import ("fmt"; "golang.org/x/crypto/bcrypt"); func main() { h,_:=bcrypt.GenerateFromPassword([]byte("'$$pass'"),10); fmt.Println(string(h)) }'); \
	mysql -u$$DB_USER -p$$DB_PASSWORD jetlink -e "INSERT INTO superadmins (email,password) VALUES ('$$email','$$HASH') ON DUPLICATE KEY UPDATE password='$$HASH';"

dev:
	@which air > /dev/null 2>&1 && air || go run ./cmd/main.go

tidy:
	go mod tidy
