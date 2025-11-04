package auth

import (
    "errors"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

var (
    jwtSecret  = []byte(getEnv("JWT_SECRET", "development-insecure-secret-change-me"))
    jwtIssuer  = getEnv("JWT_ISSUER", "task-management-api")
    jwtAudience = getEnv("JWT_AUDIENCE", "task-management-clients")
)

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

// Claims represents the JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for the given user
func GenerateToken(userID, username string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    jwtIssuer,
            Audience:  jwt.ClaimStrings{jwtAudience},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}

		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        // Validate issuer and audience
        if claims.Issuer != jwtIssuer {
            return nil, errors.New("invalid token issuer")
        }
        // Manually check audience for compatibility with jwt v5 types
        audValid := false
        for _, aud := range claims.Audience {
            if aud == jwtAudience {
                audValid = true
                break
            }
        }
        if !audValid {
            return nil, errors.New("invalid token audience")
        }
        return claims, nil
    }

	return nil, errors.New("invalid token")
}
