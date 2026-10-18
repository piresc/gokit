package auth_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/piresc/gokit/auth"
)

func genRSA(t *testing.T) (*rsa.PrivateKey, []byte, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return key, privPEM, pubPEM
}

func TestLoadRSAPrivateKey_PKCS1PEM(t *testing.T) {
	key, privPEM, _ := genRSA(t)
	got, err := auth.LoadRSAPrivateKey(string(privPEM), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(key) {
		t.Fatal("loaded key differs")
	}
}

func TestLoadRSAPrivateKey_PKCS8PEM(t *testing.T) {
	key, _, _ := genRSA(t)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	p := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	got, err := auth.LoadRSAPrivateKey(string(p), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(key) {
		t.Fatal("loaded key differs")
	}
}

func TestLoadRSAPrivateKey_EscapedNewlines(t *testing.T) {
	key, privPEM, _ := genRSA(t)
	escaped := strings.ReplaceAll(string(privPEM), "\n", `\n`)
	got, err := auth.LoadRSAPrivateKey(escaped, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(key) {
		t.Fatal("loaded key differs")
	}
}

func TestLoadRSAPrivateKey_FromFile(t *testing.T) {
	key, privPEM, _ := genRSA(t)
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, privPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := auth.LoadRSAPrivateKey("", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(key) {
		t.Fatal("loaded key differs")
	}
}

func TestLoadRSAPrivateKey_PEMWinsOverPath(t *testing.T) {
	_, privPEM, _ := genRSA(t)
	if _, err := auth.LoadRSAPrivateKey(string(privPEM), "/nonexistent/key.pem"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRSAPrivateKey_Empty(t *testing.T) {
	_, err := auth.LoadRSAPrivateKey("  ", "")
	if err == nil || !strings.Contains(err.Error(), "no RSA private key configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRSAPrivateKey_Garbage(t *testing.T) {
	if _, err := auth.LoadRSAPrivateKey("garbage", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRSAPublicKey_PKIX(t *testing.T) {
	key, _, pubPEM := genRSA(t)
	got, err := auth.LoadRSAPublicKey(string(pubPEM), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(&key.PublicKey) {
		t.Fatal("loaded key differs")
	}
}

func TestLoadRSAPublicKey_Empty(t *testing.T) {
	_, err := auth.LoadRSAPublicKey("", "")
	if err == nil || !strings.Contains(err.Error(), "no RSA public key configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRSAPublicKey_NotRSA(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&ec.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	p := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if _, err := auth.LoadRSAPublicKey(string(p), ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestRSASigner_SignVerifyRoundTrip(t *testing.T) {
	key, _, _ := genRSA(t)
	signer := auth.NewRSASigner(key, "user-service", 15*time.Minute)
	token, jti, expiresAt, err := signer.Sign("u-1", "a@b.c", "investor")
	if err != nil {
		t.Fatal(err)
	}
	if jti == "" {
		t.Fatal("empty jti")
	}
	claims, err := auth.NewRSAVerifier(&key.PublicKey).Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "u-1" || claims.Email != "a@b.c" || claims.Role != "investor" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.ID != jti || claims.Issuer != "user-service" {
		t.Fatalf("unexpected jti/issuer: %+v", claims)
	}
	if d := claims.ExpiresAt.Time.Sub(expiresAt); d > time.Second || d < -time.Second {
		t.Fatalf("exp mismatch: %v", d)
	}
}

func TestRSAVerifier_Expired(t *testing.T) {
	key, _, _ := genRSA(t)
	token, _, _, err := auth.NewRSASigner(key, "iss", -time.Minute).Sign("u", "e", "r")
	if err != nil {
		t.Fatal(err)
	}
	_, err = auth.NewRSAVerifier(&key.PublicKey).Verify(token)
	if !errors.Is(err, auth.ErrTokenExpired) {
		t.Fatalf("got %v", err)
	}
	if errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatal("expired must not match invalid")
	}
}

func TestRSAVerifier_WrongKey(t *testing.T) {
	key, _, _ := genRSA(t)
	other, _, _ := genRSA(t)
	token, _, _, err := auth.NewRSASigner(key, "iss", time.Minute).Sign("u", "e", "r")
	if err != nil {
		t.Fatal(err)
	}
	_, err = auth.NewRSAVerifier(&other.PublicKey).Verify(token)
	if !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestRSAVerifier_RejectsHS256AlgConfusion(t *testing.T) {
	key, _, pubPEM := genRSA(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "u",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	})
	s, err := tok.SignedString(pubPEM)
	if err != nil {
		t.Fatal(err)
	}
	_, err = auth.NewRSAVerifier(&key.PublicKey).Verify(s)
	if !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestRSAVerifier_Garbage(t *testing.T) {
	key, _, _ := genRSA(t)
	_, err := auth.NewRSAVerifier(&key.PublicKey).Verify("not.a.token")
	if !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("got %v", err)
	}
}

func signWith(t *testing.T, m jwt.SigningMethod, key *rsa.PrivateKey, c jwt.RegisteredClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(m, c).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRSAVerifier_MissingExp(t *testing.T) {
	key, _, _ := genRSA(t)
	s := signWith(t, jwt.SigningMethodRS256, key, jwt.RegisteredClaims{Subject: "u"})
	_, err := auth.NewRSAVerifier(&key.PublicKey).Verify(s)
	if !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestRSAVerifier_MissingSubject(t *testing.T) {
	key, _, _ := genRSA(t)
	s := signWith(t, jwt.SigningMethodRS256, key, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	})
	_, err := auth.NewRSAVerifier(&key.PublicKey).Verify(s)
	if !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestRSAVerifier_RS512Rejected(t *testing.T) {
	key, _, _ := genRSA(t)
	s := signWith(t, jwt.SigningMethodRS512, key, jwt.RegisteredClaims{
		Subject:   "u",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	})
	_, err := auth.NewRSAVerifier(&key.PublicKey).Verify(s)
	if !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("got %v", err)
	}
}
