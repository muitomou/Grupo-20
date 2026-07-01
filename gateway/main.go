package main

import (
	"context"
	"flag"
	"log"
	"net"
	"sync"
	"time"

	pb "distrieats/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type session struct {
	datanodeAddr string
	lastUsed     time.Time
}

type server struct {
	pb.UnimplementedOrderGatewayServer
	brokerAddr string
	sessions   sync.Map
}

func (s *server) cleanupSessions() {
	for {
		time.Sleep(1 * time.Minute)
		now := time.Now()
		s.sessions.Range(func(key, value interface{}) bool {
			sess := value.(session)
			if now.Sub(sess.lastUsed) > 5*time.Minute {
				s.sessions.Delete(key)
			}
			return true
		})
	}
}

func (s *server) CrearPedido(ctx context.Context, req *pb.CrearPedidoRequest) (*pb.CrearPedidoResponse, error) {
	conn, err := grpc.Dial(s.brokerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewOrderGatewayClient(conn)
	res, err := client.CrearPedido(ctx, req)
	if err != nil {
		return nil, err
	}

	s.sessions.Store(req.ClienteId, session{
		datanodeAddr: res.DatanodeId,
		lastUsed:     time.Now(),
	})

	return res, nil
}

func (s *server) ConsultarEstado(ctx context.Context, req *pb.ConsultarEstadoRequest) (*pb.ConsultarEstadoResponse, error) {
	val, ok := s.sessions.Load(req.ClienteId)
	if ok {
		sess := val.(session)
		s.sessions.Store(req.ClienteId, session{
			datanodeAddr: sess.datanodeAddr,
			lastUsed:     time.Now(),
		})

		conn, err := grpc.Dial(sess.datanodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			defer conn.Close()
			dnClient := pb.NewDatanodeServiceClient(conn)
			return dnClient.ConsultarEstado(ctx, req)
		}
	}

	conn, err := grpc.Dial(s.brokerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewOrderGatewayClient(conn)
	return client.ConsultarEstado(ctx, req)
}

func main() {
	port := flag.String("port", "8080", "")
	broker := flag.String("broker", "localhost:9090", "")
	flag.Parse()

	lis, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Fallo: %v", err)
	}

	srv := &server{
		brokerAddr: *broker,
	}

	go srv.cleanupSessions()

	s := grpc.NewServer()
	pb.RegisterOrderGatewayServer(s, srv)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fallo en serve: %v", err)
	}
}
