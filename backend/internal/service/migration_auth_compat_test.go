package service

import (
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// legacyProductionJWTClaims is the released GotoCC access-token wire format.
// Keep this fixture independent from JWTClaims so a candidate field/tag change
// cannot make a compatibility test pass by changing both writer and reader.
type legacyProductionJWTClaims struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenVersion int64  `json:"token_version"`
	SessionID    string `json:"sid,omitempty"`
	BindingHash  string `json:"bnd,omitempty"`
	jwt.RegisteredClaims
}

func TestPlusCandidateReadsLegacyProductionJWT(t *testing.T) {
	const secret = "migration-fixture-secret-32-bytes"
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, legacyProductionJWTClaims{
		UserID:       41,
		Email:        "migration@example.test",
		Role:         RoleUser,
		TokenVersion: 9,
		SessionID:    "legacy-family",
		BindingHash:  "legacy-binding",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Unix(1700000000, 0)),
			NotBefore: jwt.NewNumericDate(time.Unix(1700000000, 0)),
			ExpiresAt: jwt.NewNumericDate(time.Unix(4102444800, 0)),
		},
	})
	wireToken, err := legacy.SignedString([]byte(secret))
	require.NoError(t, err)

	candidate := NewAuthService(nil, nil, nil, nil, &config.Config{JWT: config.JWTConfig{Secret: secret}}, nil, nil, nil, nil, nil, nil, nil, nil)
	claims, err := candidate.ValidateToken(wireToken)
	require.NoError(t, err)
	require.Equal(t, int64(41), claims.UserID)
	require.Equal(t, "migration@example.test", claims.Email)
	require.Equal(t, RoleUser, claims.Role)
	require.Equal(t, int64(9), claims.TokenVersion)
	require.Equal(t, "legacy-family", claims.SessionID)
	require.Equal(t, "legacy-binding", claims.BindingHash)
}

func TestPlusCandidateReadsPreBindingLegacyJWT(t *testing.T) {
	const secret = "migration-fixture-secret-32-bytes"
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, legacyProductionJWTClaims{
		UserID:       42,
		Email:        "old-session@example.test",
		Role:         RoleUser,
		TokenVersion: 3,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(4102444800, 0)),
		},
	})
	wireToken, err := legacy.SignedString([]byte(secret))
	require.NoError(t, err)

	candidate := NewAuthService(nil, nil, nil, nil, &config.Config{JWT: config.JWTConfig{Secret: secret}}, nil, nil, nil, nil, nil, nil, nil, nil)
	claims, err := candidate.ValidateToken(wireToken)
	require.NoError(t, err)
	require.Empty(t, claims.SessionID)
	require.Empty(t, claims.BindingHash)
}
