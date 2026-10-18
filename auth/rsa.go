package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrTokenExpired is returned when a token's exp claim is in the past.
	ErrTokenExpired = errors.New("auth: token expired")
	// ErrTokenInvalid is returned for any other verification failure.
	ErrTokenInvalid = errors.New("auth: token invalid")
)

// RSAClaims are the claims carried by RS256 tokens. Subject is the user ID
// and ID is the token's jti.
type RSAClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// readPEM returns PEM bytes from pemValue (literal "\n" sequences expanded) or,
// when empty, from the file at path. Returns nil, nil when both are empty.
func readPEM(pemValue, path string) ([]byte, error) {
	v := strings.TrimSpace(strings.ReplaceAll(pemValue, `\n`, "\n"))
	if v != "" {
		return []byte(v), nil
	}
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("auth: read key file: %w", err)
	}
	return data, nil
}

// LoadRSAPrivateKey loads an RSA private key (PKCS#1 or PKCS#8) from pemValue,
// falling back to the file at path when pemValue is empty.
func LoadRSAPrivateKey(pemValue, path string) (*rsa.PrivateKey, error) {
	data, err := readPEM(pemValue, path)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errors.New("auth: no RSA private key configured")
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("auth: invalid PEM for RSA private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("auth: parse RSA private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("auth: private key is not RSA")
	}
	return key, nil
}

// LoadRSAPublicKey loads an RSA public key (PKIX) from pemValue, falling back
// to the file at path when pemValue is empty.
func LoadRSAPublicKey(pemValue, path string) (*rsa.PublicKey, error) {
	data, err := readPEM(pemValue, path)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errors.New("auth: no RSA public key configured")
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("auth: invalid PEM for RSA public key")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("auth: parse RSA public key: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("auth: public key is not RSA")
	}
	return key, nil
}

// RSASigner issues RS256 tokens.
type RSASigner struct {
	key    *rsa.PrivateKey
	issuer string
	ttl    time.Duration
}

// NewRSASigner creates a signer with the given key, issuer and token lifetime.
func NewRSASigner(key *rsa.PrivateKey, issuer string, ttl time.Duration) *RSASigner {
	return &RSASigner{key: key, issuer: issuer, ttl: ttl}
}

// Sign issues a token for the user and returns it with its jti and expiry.
func (s *RSASigner) Sign(userID, email, role string) (token, jti string, expiresAt time.Time, err error) {
	if s.key == nil {
		return "", "", time.Time{}, errors.New("auth: no RSA private key configured")
	}
	now := time.Now()
	expiresAt = now.Add(s.ttl)
	jti = uuid.NewString()

	claims := &RSAClaims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token, err = jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.key)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return token, jti, expiresAt, nil
}

// RSAVerifier verifies RS256 tokens.
type RSAVerifier struct {
	key *rsa.PublicKey
}

// NewRSAVerifier creates a verifier for the given public key.
func NewRSAVerifier(key *rsa.PublicKey) *RSAVerifier {
	return &RSAVerifier{key: key}
}

// Verify parses and validates a token. Errors wrap ErrTokenExpired or
// ErrTokenInvalid.
func (v *RSAVerifier) Verify(token string) (*RSAClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &RSAClaims{}, func(*jwt.Token) (interface{}, error) {
		if v.key == nil {
			return nil, errors.New("no public key configured")
		}
		return v.key, nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithExpirationRequired())
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %v", ErrTokenExpired, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	claims, ok := parsed.Claims.(*RSAClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("%w: invalid claims", ErrTokenInvalid)
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: missing subject", ErrTokenInvalid)
	}
	return claims, nil
}
