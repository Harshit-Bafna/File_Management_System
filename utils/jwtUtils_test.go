package utils

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestGenerateJWT(t *testing.T) {
	claims := &Claims{
		Email: "test@example.com",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		},
	}

	token, err := GenerateJWT(claims)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestParseToken(t *testing.T) {
	claims := &Claims{
		Email: "test@example.com",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		},
	}

	token, err := GenerateJWT(claims)
	assert.NoError(t, err)

	parsedClaims, err := ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, claims.Email, parsedClaims.Email)
}
