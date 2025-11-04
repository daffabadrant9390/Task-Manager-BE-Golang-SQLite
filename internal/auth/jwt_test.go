package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateToken(t *testing.T) {
	token, err := GenerateToken("u-1", "alice")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, "u-1", claims.UserID)
	require.Equal(t, "alice", claims.Username)
}

func TestValidateToken_Invalid(t *testing.T) {
	_, err := ValidateToken("invalid.token")
	require.Error(t, err)
}


