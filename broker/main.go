package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync/atomic"

	pb "distrieats/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type server struct {
	pb.UnimplementedOrderGatewayServer
	datanodes []string
	nextDN    uint64
}

func (s *server) callWithRetry(call func(string) (interface{}, error)) (interface{}, string, error) {
	startIdx := atomic.AddUint64(&s.nextDN, 1)
	for i := 0; i < len(s.datanodes); i++ {
		idx := (int(startIdx) + i) % len(s.datanodes)
		dn := s.datanodes[idx]
		res, err := call(dn)
		if err == nil {
			return res, dn, nil
		}
		log.Printf("Fallo al contactar %s, derivando a otro nodo...", dn)
	}
	return nil, "", fmt.Errorf("todos los datanodes caidos")
}

func (s *server) CrearPedido(ctx context.Context, req *pb.CrearPedidoRequest) (*pb.CrearPedidoResponse, error) {
	res, dn, err := s.callWithRetry(func(target string) (interface{}, error) {
		conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := pb.NewDatanodeServiceClient(conn)
		return client.CrearPedido(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	r := res.(*pb.CrearPedidoResponse)
	r.DatanodeId = dn
	return r, nil
}

func (s *server) ConsultarEstado(ctx context.Context, req *pb.ConsultarEstadoRequest) (*pb.ConsultarEstadoResponse, error) {
	res, _, err := s.callWithRetry(func(target string) (interface{}, error) {
		conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := pb.NewDatanodeServiceClient(conn)
		return client.ConsultarEstado(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return res.(*pb.ConsultarEstadoResponse), nil
}

func (s *server) EnviarActualizacion(ctx context.Context, req *pb.UpdateOrderRequest) (*pb.UpdateOrderResponse, error) {
	_, _, err := s.callWithRetry(func(target string) (interface{}, error) {
		conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := pb.NewDatanodeServiceClient(conn)
		return client.UpdateOrder(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateOrderResponse{Success: true}, nil
}

func main() {
	port := flag.String("port", "9090", "")
	datanodesFlag := flag.String("datanodes", "localhost:50051,localhost:50052,localhost:50053", "")
	flag.Parse()

	datanodes := strings.Split(*datanodesFlag, ",")

	lis, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Fallo: %v", err)
	}

	srv := &server{
		datanodes: datanodes,
	}

	s := grpc.NewServer()
	pb.RegisterOrderGatewayServer(s, srv)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fallo en serve: %v", err)
	}
}
