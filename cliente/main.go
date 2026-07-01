package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pb "distrieats/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	gatewayAddr := flag.String("gateway", "localhost:8080", "")
	clienteID := flag.String("id", "Cliente-1", "")
	flag.Parse()

	conn, err := grpc.Dial(*gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Fallo: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderGatewayClient(conn)
	reqID := fmt.Sprintf("Req-%s-%d", *clienteID, time.Now().UnixNano())
	pedidoID := fmt.Sprintf("Ped-%s-%d", *clienteID, time.Now().Unix())

	crearReq := &pb.CrearPedidoRequest{
		RequestId: reqID,
		ClienteId: *clienteID,
		PedidoId:  pedidoID,
	}

	crearRes, err := client.CrearPedido(context.Background(), crearReq)
	if err != nil || !crearRes.Success {
		log.Fatalf("Error creando pedido: %v", err)
	}

	log.Printf("Pedido %s creado exitosamente (Asignado a %s)", pedidoID, crearRes.DatanodeId)

	consultarReq := &pb.ConsultarEstadoRequest{
		ClienteId: *clienteID,
		PedidoId:  pedidoID,
	}

	consultarRes, err := client.ConsultarEstado(context.Background(), consultarReq)
	if err != nil {
		log.Fatalf("Error consultando estado: %v", err)
	}

	if consultarRes.Estado != "No Encontrado" {
		log.Printf("\n=== VALIDACION READ YOUR WRITES EXITOSA ===\nCliente %s (Pedido: %s): Estado encontrado -> %s\nAfinidad de sesion confirmada.\n===========================================\n", *clienteID, pedidoID, consultarRes.Estado)
		client.ReportarRYW(context.Background(), &pb.RYWRequest{
			ClientId:   *clienteID,
			PedidoId:   pedidoID,
			DatanodeId: crearRes.DatanodeId,
		})
	} else {
		log.Fatalf("Falló READ YOUR WRITES: El pedido %s no fue encontrado inmediatamente despues de crearlo.", pedidoID)
	}

	client.ReportarTermino(context.Background(), &pb.ClientDoneRequest{
		ClientId: *clienteID,
	})
}
