package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"log"
	"math/big"
	"time"

	pb "github.com/hexfusion/spore/proto/pb/p2p"
	"google.golang.org/protobuf/proto"
)

func HashPublicKey(pub ed25519.PublicKey) string {
	hash := sha256.Sum256(pub)
	return hex.EncodeToString(hash[:])
}

func SignTrustedPeer(tp *pb.TrustedPeer, priv ed25519.PrivateKey) error {
	copy := proto.Clone(tp).(*pb.TrustedPeer)
	copy.Signature = nil
	data, err := proto.Marshal(copy)
	if err != nil {
		return err
	}
	tp.Signature = ed25519.Sign(priv, data)
	return nil
}

func VerifyTrustedPeer(tp *pb.TrustedPeer) error {
	if len(tp.PublicKey) != ed25519.PublicKeySize {
		return errors.New("invalid public key length")
	}
	pub := ed25519.PublicKey(tp.PublicKey)

	expectedID := HashPublicKey(pub)
	if tp.Id != expectedID {
		return errors.New("node ID mismatch")
	}

	copy := proto.Clone(tp).(*pb.TrustedPeer)
	copy.Signature = nil
	data, err := proto.Marshal(copy)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, data, tp.Signature) {
		return errors.New("signature verification failed")
	}
	return nil
}

func encodeToPEM(certDER []byte, key *ecdsa.PrivateKey) ([]byte, []byte, error) {
	certBuf := new(bytes.Buffer)
	pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	keyBuf := new(bytes.Buffer)
	pem.Encode(keyBuf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certBuf.Bytes(), keyBuf.Bytes(), nil
}

func generateInsecureTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"spore"},
		Certificates:       []tls.Certificate{generateSelfSignedCert()},
	}
}

func generateSelfSignedCert() tls.Certificate {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-1 * time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour), // Valid for 1 day

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("failed to create cert: %v", err)
	}

	certPEM, keyPEM, err := encodeToPEM(derBytes, priv)
	if err != nil {
		log.Fatalf("failed to encode cert/key: %v", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("failed to load cert: %v", err)
	}
	return cert
}
