package auth

import (
	"chirpy/internal/logger"
	"errors"
	"net/http"
	"strings"
	"time"
	"crypto/rand"
	"encoding/hex"
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// HashPassword creates an Argon2id hash of the plain-text password.
func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

// CheckPasswordHash compares a plain-text password with a hash.
func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	return match, err
}


func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer: "chirpy",
		Subject: userID.String(),
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(tokenSecret))
}
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// ---- 1. Verify signing method ----
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Logger.Errorw("invalid signing method",
				"expected", "HMAC",
				"got", t.Header["alg"],
			)
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(tokenSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		// ---- 2. Log parsing/validation errors ----
		// jwt-go classifies many errors – we log the root cause
		logger.Logger.Errorw("jwt parsing failed",
			"error", err,
			"token_preview", TruncateToken(tokenString),
		)
		return uuid.Nil, err
	}

	if !token.Valid {
		logger.Logger.Infow("jwt token is invalid (expired, malformed, etc.)",
			"token_preview", TruncateToken(tokenString),
		)
		return uuid.Nil, jwt.ErrTokenExpired
	}

	// ---- 3. Extract Subject (user ID) ----
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		logger.Logger.Errorw("subject claim is not a valid UUID",
			"subject", claims.Subject,
			"error", err,
		)
		return uuid.Nil, err
	}

	return userID, nil
}

// truncateToken returns first 12 chars of the token for safe logging.
func TruncateToken(token string) string {
	if len(token) > 12 {
		return token[:12] + "..."
	}
	return token
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is missing")
	}

    // Split on whitespace, ignore multiple spaces
	parts := strings.Fields(authHeader)
	if len(parts) < 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("authorization header must be in 'Bearer <token>' format")
	}

	token := parts[1]
	if token == "" {
		return "", errors.New("token is empty")
	}

	return token, nil
}

func MakeRefreshToken() (string, error) {
	b:= make([]byte, 32)

	_, err := rand.Read(b)
	if err!= nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "ApiKey" {
		return "", errors.New("invalid Authorization format")
	}

	key := strings.TrimSpace(parts[1])
	if key == "" {
		return "", errors.New("empty API key")
	}

	return key, nil
}