package testkit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

// NewRSAKeyPair generates a 2048-bit RSA key pair for tests. The private key
// is returned PKCS#1 PEM-encoded and the public key PKIX PEM-encoded.
func NewRSAKeyPair(t testing.TB) (priv *rsa.PrivateKey, privPEM, pubPEM []byte) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("testkit: generate RSA key: %v", err)
	}

	privPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("testkit: marshal RSA public key: %v", err)
	}
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	return priv, privPEM, pubPEM
}
