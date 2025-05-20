package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"

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
