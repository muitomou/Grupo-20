package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pb "distrieats/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type server struct {
	pb.UnimplementedOrderGatewayServer
	datanodes     []string
	nextDN        uint64
	finishedCount int32
	expectedCount int32
	mu            sync.Mutex
	rywLogs       []string
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

func (s *server) ReportarTermino(ctx context.Context, req *pb.ClientDoneRequest) (*pb.ClientDoneResponse, error) {
	val := atomic.AddInt32(&s.finishedCount, 1)
	log.Printf("ReportarTermino recibido de %s (Total: %d/%d)", req.ClientId, val, s.expectedCount)
	if val == s.expectedCount {
		log.Printf("Todas las entidades han terminado. Iniciando periodo de gracia de 16 segundos...")
		go s.triggerGracePeriodAndAudit()
	}
	return &pb.ClientDoneResponse{Success: true}, nil
}

func (s *server) ReportarRYW(ctx context.Context, req *pb.RYWRequest) (*pb.RYWResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := fmt.Sprintf("- %s ( %s) : Validacion Exitosa en %s ( Afinidad de sesion confirmada ) .", req.ClientId, req.PedidoId, req.DatanodeId)
	s.rywLogs = append(s.rywLogs, msg)
	return &pb.RYWResponse{Success: true}, nil
}

func (s *server) triggerGracePeriodAndAudit() {
	var wg sync.WaitGroup
	for _, dn := range s.datanodes {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return
			}
			defer conn.Close()
			client := pb.NewDatanodeServiceClient(conn)
			client.SignalGracePeriod(context.Background(), &pb.GraceRequest{})
		}(dn)
	}
	wg.Wait()

	time.Sleep(16 * time.Second)

	var mu sync.Mutex
	states := make(map[string]string)
	for _, dn := range s.datanodes {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return
			}
			defer conn.Close()
			client := pb.NewDatanodeServiceClient(conn)
			res, err := client.GetFinalState(ctx, &pb.StateRequest{})
			if err == nil {
				mu.Lock()
				states[target] = res.StateData
				mu.Unlock()
			}
		}(dn)
	}
	wg.Wait()

	reportData := "=== REPORTE FINAL : DISTRIEATS ===\n[ ESTADO GLOBAL DE PEDIDOS -\n"
	var finalState string
	convergence := true
	for target, st := range states {
		if finalState == "" {
			finalState = st
		} else if finalState != st {
			convergence = false
			log.Printf("Divergencia detectada con el nodo %s", target)
		}
	}
	
	if finalState != "" {
		reportData += finalState
	}

	if convergence {
		reportData += "Convergencia Alcanzada ]\n"
	} else {
		reportData += "Divergencia Detectada ]\n"
	}

	reportData += "\n[ AUDITORIA READ YOUR WRITES ]\n"
	s.mu.Lock()
	for _, logMsg := range s.rywLogs {
		reportData += logMsg + "\n"
	}
	s.mu.Unlock()
	reportData += "================================="

	err := os.WriteFile("Reporte.txt", []byte(reportData), 0644)
	if err != nil {
		log.Printf("Error escribiendo Reporte.txt: %v", err)
	} else {
		log.Printf("Reporte.txt generado exitosamente en el contenedor del Broker.")
	}
}

func main() {
	port := flag.String("port", "9090", "")
	datanodesFlag := flag.String("datanodes", "localhost:50051,localhost:50052,localhost:50053", "")
	expectedCount := flag.Int("expected", 4, "")
	flag.Parse()

	datanodes := strings.Split(*datanodesFlag, ",")

	lis, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Fallo: %v", err)
	}

	srv := &server{
		datanodes:     datanodes,
		expectedCount: int32(*expectedCount),
	}

	s := grpc.NewServer()
	pb.RegisterOrderGatewayServer(s, srv)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fallo en serve: %v", err)
	}
}
