package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func ECDSAGenerateKeys() (err error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	privateBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to converts an EC private key to SEC 1: %w", err)
	}

	privateFile, err := os.Create("ecdsa_private.pem")
	if err != nil {
		return fmt.Errorf("failed to open private file: %w", err)
	}

	defer func() {
		if cErr := privateFile.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close privare file: %w", err, cErr)
		}
	}()

	if err = pem.Encode(privateFile, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateBytes,
	}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	publicBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to converts a public key to PKIX: %w", err)
	}

	publicFile, err := os.Create("ecdsa_public.pem")
	if err != nil {
		return fmt.Errorf("failed to open public file: %w", err)
	}

	defer func() {
		if cErr := publicFile.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close public file: %w", err, cErr)
		}
	}()

	if err = pem.Encode(publicFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicBytes,
	}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

func MustECDSAGenerateKeys() {
	if err := ECDSAGenerateKeys(); err != nil {
		panic(err)
	}
}
