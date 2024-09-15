package utils

import (
	"authentication/config"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

var secretKey = []byte("your_secret_key")

type Claims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

func GenerateJWT(claims *Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, err
	}
	return claims, nil
}

func GetUserIdFromToken(tokenString string) (int, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %v", err)
	}

	email := claims.Email
	var userID int
	err = config.DB.QueryRow("SELECT id FROM users WHERE email=$1", email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve user ID: %v", err)
	}

	return userID, nil
}
