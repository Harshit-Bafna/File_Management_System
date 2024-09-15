package models

import (
	"authentication/config"
	"fmt"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func UserExists(email string) bool {
	var exists bool
	err := config.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE email=$1)", email).Scan(&exists)
	if err != nil {
		fmt.Println("Error checking user existence:", err)
		return false
	}
	return exists
}

func CreateUser(email, hashedPassword string) error {
	_, err := config.DB.Exec("INSERT INTO users (email, password) VALUES ($1, $2)", email, hashedPassword)
	return err
}

func GetPasswordByEmail(email string) (string, error) {
	var password string
	err := config.DB.QueryRow("SELECT password FROM users WHERE email=$1", email).Scan(&password)
	return password, err
}
