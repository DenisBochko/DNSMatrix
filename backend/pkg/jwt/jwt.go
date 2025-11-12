package jwt

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenOption func(claims jwt.MapClaims)

func WithClaim(key string, value any) TokenOption {
	return func(claims jwt.MapClaims) {
		claims[key] = value
	}
}

// LoadECDSAPrivateKey loads the ECDSA private key (P-256)
func LoadECDSAPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	privateKey, err := jwt.ParseECPrivateKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EC private key: %w", err)
	}

	return privateKey, nil
}

// LoadECDSAPublicKey loads the ECDSA public key
func LoadECDSAPublicKey(path string) (*ecdsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	publicKey, err := jwt.ParseECPublicKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EC public key: %w", err)
	}

	return publicKey, nil
}

// NewToken uses an Asymmetric encryption algorithm
func NewToken(privateKey *ecdsa.PrivateKey, duration time.Duration, opts ...TokenOption) (string, error) {
	token := jwt.New(jwt.SigningMethodES256)

	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().UTC().Add(duration).Unix()

	for _, opt := range opts {
		opt(claims)
	}

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken verifies the JWT token using a public key
func ValidateToken(tokenString string, publicKey *ecdsa.PublicKey) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
