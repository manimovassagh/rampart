package signing

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

const rsaKeyBits = 4096

// KeyPair holds the RSA key pair and its computed Key ID.
type KeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KID        string
}

// LoadOrGenerate loads an RSA private key from the given PEM file path.
// If the file does not exist, it generates a new 4096-bit key pair,
// writes it to the file, and returns the key pair.
func LoadOrGenerate(path string) (*KeyPair, error) {
	clean := filepath.Clean(path)
	data, err := os.ReadFile(clean)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading signing key %q: %w", clean, err)
		}
		return generate(clean)
	}
	return parse(data)
}

// generate creates a new RSA key pair, persists it as PEM, and returns the KeyPair.
func generate(path string) (*KeyPair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, fmt.Errorf("generating RSA key: %w", err)
	}

	derBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshaling private key: %w", err)
	}

	pemBlock := &pem.Block{Type: "PRIVATE KEY", Bytes: derBytes}
	pemData := pem.EncodeToMemory(pemBlock)

	if err := os.WriteFile(path, pemData, 0o600); err != nil {
		return nil, fmt.Errorf("writing signing key to %q: %w", path, err)
	}

	kid := computeKID(&priv.PublicKey)

	return &KeyPair{
		PrivateKey: priv,
		PublicKey:  &priv.PublicKey,
		KID:        kid,
	}, nil
}

// parse decodes a PEM-encoded private key and returns the KeyPair.
func parse(data []byte) (*KeyPair, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in signing key file")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	priv, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("signing key is not RSA")
	}

	kid := computeKID(&priv.PublicKey)

	return &KeyPair{
		PrivateKey: priv,
		PublicKey:  &priv.PublicKey,
		KID:        kid,
	}, nil
}

// computeKID computes a JWK Thumbprint (RFC 7638) for the given RSA public key.
// The thumbprint is the base64url-encoded SHA-256 hash of the canonical JWK representation.
func computeKID(pub *rsa.PublicKey) string {
	// RFC 7638: canonical JSON with lexicographic member ordering for RSA: {e, kty, n}
	e := base64URLUint(big.NewInt(int64(pub.E)))
	n := base64URLUint(pub.N)
	canonical := `{"e":"` + e + `","kty":"RSA","n":"` + n + `"}`

	hash := sha256.Sum256([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// JWK returns the public key as a JWK (RFC 7517) map.
func (kp *KeyPair) JWK() map[string]string {
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kp.KID,
		"n":   base64URLUint(kp.PublicKey.N),
		"e":   base64URLUint(big.NewInt(int64(kp.PublicKey.E))),
	}
}

// JWKSResponse returns the JWKS JSON structure containing this key.
func (kp *KeyPair) JWKSResponse() ([]byte, error) {
	jwks := struct {
		Keys []map[string]string `json:"keys"`
	}{
		Keys: []map[string]string{kp.JWK()},
	}
	return json.Marshal(jwks)
}

// base64URLUint encodes a big.Int as unpadded base64url per RFC 7518 §2.
func base64URLUint(n *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(n.Bytes())
}
