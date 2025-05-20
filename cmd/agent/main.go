package main

import (
	"crypto/ed25519"
	"fmt"
	"log"
	"net"

	pb "github.com/hexfusion/spore/proto/pb/p2p"
	"google.golang.org/protobuf/proto"
)

const addrA = ":9999"
const addrB = ":10000"

func main() {
	go runPeerB()

	runPeerA()
}

func runPeerA() {
	pub, priv, _ := ed25519.GenerateKey(nil)

	peer := &pb.TrustedPeer{
		Address:      "127.0.0.1" + addrA,
		PublicKey:    pub,
		Capabilities: []string{"oci-host"},
	}
	peer.Id = HashPublicKey(pub)
	_ = SignTrustedPeer(peer, priv)

	msg := &pb.Message{
		Message: &pb.Message_PeerList{
			PeerList: &pb.PeerList{
				Peers: []*pb.TrustedPeer{peer},
			},
		},
	}

	buf, _ := proto.Marshal(msg)

	conn, _ := net.Dial("udp", addrB)
	defer conn.Close()
	conn.Write(buf)
	fmt.Println("Peer A: sent TrustedPeer to B")
}

func runPeerB() {
	addr, _ := net.ResolveUDPAddr("udp", addrB)
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	buf := make([]byte, 2048)
	n, _, _ := conn.ReadFrom(buf)
	fmt.Println("Peer B: received message")

	var msg pb.Message
	if err := proto.Unmarshal(buf[:n], &msg); err != nil {
		log.Fatal("Failed to unmarshal:", err)
	}

	switch m := msg.Message.(type) {
	case *pb.Message_PeerList:
		for _, peer := range m.PeerList.Peers {
			if err := VerifyTrustedPeer(peer); err != nil {
				fmt.Println("❌ Invalid peer:", err)
			} else {
				fmt.Printf("✅ Trusted peer: %s (%s)\n", peer.Id, peer.Address)
			}
		}
	default:
		fmt.Println("Unhandled message type")
	}
}
