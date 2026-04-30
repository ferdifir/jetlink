package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/ferdifir/jetlink/internal/config"
	"github.com/ferdifir/jetlink/internal/database"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	email := "admin@jetlink.my.id"
	password := "jet54@dmin"

	if len(os.Args) >= 3 {
		email = os.Args[1]
		password = os.Args[2]
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(
		`INSERT INTO superadmins (email, password) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE password=VALUES(password)`,
		email, string(hashed),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Superadmin seeded: %s\n", email)
}
