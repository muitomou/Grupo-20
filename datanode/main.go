package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	pb "distrieats/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	statePriority = map[string]int{
		"Recibido":   1,
		"Preparando": 2,
		"En Camino":  3,
		"Entregado":  4,
		"Cancelado":  5,
	}
)

type server struct {
	pb.UnimplementedDatanodeServiceServer
	nodeID string
	peers  []string
	mu     sync.RWMutex
	data   map[string]*pb.PedidoData
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func mergeClocks(clock1, clock2 map[string]int32) map[string]int32 {
	merged := make(map[string]int32)
	for k, v := range clock1 {
		merged[k] = v
	}
	for k, v := range clock2 {
		merged[k] = max(merged[k], v)
	}
	return merged
}

func resolveState(currentState, newState string) string {
	pCurrent := statePriority[currentState]
	pNew := statePriority[newState]
	if pNew > pCurrent {
		return newState
	}
	return currentState
}

func (s *server) UpdateOrder(ctx context.Context, req *pb.UpdateOrderRequest) (*pb.UpdateOrderResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.data[req.PedidoId]
	if !exists {
		s.data[req.PedidoId] = &pb.PedidoData{
			Estado:      req.Estado,
			VectorClock: req.VectorClock,
		}
		log.Printf("Nuevo pedido %s guardado: %s", req.PedidoId, req.Estado)
		return &pb.UpdateOrderResponse{Success: true}, nil
	}

	mergedClock := mergeClocks(existing.VectorClock, req.VectorClock)
	resolvedState := resolveState(existing.Estado, req.Estado)

	s.data[req.PedidoId] = &pb.PedidoData{
		Estado:      resolvedState,
		VectorClock: mergedClock,
	}

	log.Printf("Pedido %s actualizado. Estado final: %s", req.PedidoId, resolvedState)
	return &pb.UpdateOrderResponse{Success: true}, nil
}

func (s *server) ConsultarEstado(ctx context.Context, req *pb.ConsultarEstadoRequest) (*pb.ConsultarEstadoResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pedido, exists := s.data[req.PedidoId]
	if !exists {
		return &pb.ConsultarEstadoResponse{PedidoId: req.PedidoId, Estado: "No Encontrado"}, nil
	}

	return &pb.ConsultarEstadoResponse{
		PedidoId: req.PedidoId,
		Estado:   pedido.Estado,
	}, nil
}

func (s *server) GossipSync(ctx context.Context, req *pb.GossipRequest) (*pb.GossipResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for pedidoID, incomingData := range req.EstadoPedidos {
		existing, exists := s.data[pedidoID]
		if !exists {
			s.data[pedidoID] = incomingData
		} else {
			s.data[pedidoID].VectorClock = mergeClocks(existing.VectorClock, incomingData.VectorClock)
			s.data[pedidoID].Estado = resolveState(existing.Estado, incomingData.Estado)
		}
	}

	return &pb.GossipResponse{Success: true}, nil
}

func (s *server) startGossip() {
	if len(s.peers) == 0 {
		return
	}
	for {
		time.Sleep(5 * time.Second)
		peer := s.peers[rand.Intn(len(s.peers))]

		conn, err := grpc.Dial(peer, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			continue
		}
		client := pb.NewDatanodeServiceClient(conn)

		s.mu.RLock()
		stateCopy := make(map[string]*pb.PedidoData)
		for k, v := range s.data {
			stateCopy[k] = v
		}
		s.mu.RUnlock()

		req := &pb.GossipRequest{
			OrigenNodeId:  s.nodeID,
			EstadoPedidos: stateCopy,
		}

		_, _ = client.GossipSync(context.Background(), req)
		conn.Close()
	}
}
func (s *server) CrearPedido(ctx context.Context, req *pb.CrearPedidoRequest) (*pb.CrearPedidoResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.data[req.PedidoId]
	if !exists {
		s.data[req.PedidoId] = &pb.PedidoData{
			Estado:      "Recibido",
			VectorClock: map[string]int32{s.nodeID: 1},
		}
		log.Printf("Pedido %s creado (Recibido)", req.PedidoId)
	}
	return &pb.CrearPedidoResponse{Success: true, DatanodeId: s.nodeID}, nil
}

func main() {
	port := flag.String("port", "50051", "")
	nodeID := flag.String("id", "DN1", "")
	peersFlag := flag.String("peers", "", "")
	flag.Parse()

	peers := []string{}
	if *peersFlag != "" {
		peers = strings.Split(*peersFlag, ",")
	}

	lis, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Fallo: %v", err)
	}

	s := grpc.NewServer()
	srv := &server{
		nodeID: *nodeID,
		peers:  peers,
		data:   make(map[string]*pb.PedidoData),
	}

	pb.RegisterDatanodeServiceServer(s, srv)

	go srv.startGossip()

	log.Printf("Datanode %s escuchando en %s", *nodeID, *port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fallo en serve: %v", err)
	}
}
