package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/quic-go/quic-go"
	pb "github.com/hexfusion/spore/proto/pb/p2p"
	"google.golang.org/protobuf/proto"
)

const (
	addrB = "localhost:6121"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runPeerB(ctx)
	time.Sleep(1 * time.Second)
	
	runPeerA(ctx)
	time.Sleep(1 * time.Second)
}

func runPeerA(ctx context.Context) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"spore"},
	}

	session, err := quic.DialAddr(ctx, addrB, tlsConf, nil)
	if err != nil {
		log.Fatal("dial error:", err)
	}
	defer session.CloseWithError(0, "done")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		log.Fatal("open stream error:", err)
	}
	defer stream.Close()

	msg := &pb.Message{
		Message: &pb.Message_Ping{Ping: &pb.Ping{}},
	}
	buf, _ := proto.Marshal(msg)
	stream.Write(buf)
	fmt.Println("Peer A: sent Ping")

	// Read reply
	replyBuf := make([]byte, 1024)
	n, err := stream.Read(replyBuf)
if err != nil && err.Error() != "EOF" {
	log.Fatal("read error:", err)
}
	var reply pb.Message
	if err := proto.Unmarshal(replyBuf[:n], &reply); err == nil {
		if _, ok := reply.Message.(*pb.Message_Pong); ok {
			fmt.Println("Peer A received Pong")
		}
	}
}

func runPeerB(ctx context.Context) {
	tlsConf := generateInsecureTLSConfig()
	listener, err := quic.ListenAddr(addrB, tlsConf, nil)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		go handlePeer(ctx, conn)
	}
}

func handlePeer(ctx context.Context, conn quic.Connection) {
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		log.Println("accept stream error:", err)
		return
	}
	defer stream.Close()

	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		log.Println("read error:", err)
		return
	}

	var msg pb.Message
	if err := proto.Unmarshal(buf[:n], &msg); err != nil {
		log.Println("unmarshal error:", err)
		return
	}

	switch msg.Message.(type) {
	case *pb.Message_Ping:
		fmt.Println("Peer B received Ping — sending Pong")
		reply := &pb.Message{
			Message: &pb.Message_Pong{Pong: &pb.Pong{}},
		}
		out, _ := proto.Marshal(reply)
		stream.Write(out)
	default:
		fmt.Println("Unhandled message type")
	}
}
