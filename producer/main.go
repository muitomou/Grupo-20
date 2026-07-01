package main

import (
	"context"
	"encoding/csv"
	"flag"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	pb "distrieats/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	brokerAddr := flag.String("broker", "localhost:9090", "")
	csvPath := flag.String("csv", "pedidos.csv", "")
	flag.Parse()

	file, err := os.Open(*csvPath)
	if err != nil {
		log.Fatalf("Fallo al abrir csv: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, _ = reader.Read()

	clock := int32(0)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 4 {
			continue
		}

		pedidoID := record[0]
		estado := record[3]

		time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
		clock++

		vc := map[string]int32{"Producer": clock}
		req := &pb.UpdateOrderRequest{
			PedidoId:    pedidoID,
			Estado:      estado,
			VectorClock: vc,
		}

		var err error
		for retries := 0; retries < 20; retries++ {
			conn, dialErr := grpc.Dial(*brokerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if dialErr != nil {
				time.Sleep(3 * time.Second)
				continue
			}
			client := pb.NewOrderGatewayClient(conn)
			_, err = client.EnviarActualizacion(context.Background(), req)
			conn.Close()

			if err == nil {
				break
			}
			log.Printf("Red no lista, reintentando actualizacion %s en 3s... (%v)", pedidoID, err)
			time.Sleep(3 * time.Second)
		}

		if err != nil {
			log.Printf("Error definitivo enviando actualizacion %s: %v", pedidoID, err)
		} else {
			log.Printf("Actualizacion enviada: %s -> %s", pedidoID, estado)
		}
	}
	log.Println("Simulacion finalizada")

	conn, err := grpc.Dial(*brokerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()
		client := pb.NewOrderGatewayClient(conn)
		client.ReportarTermino(context.Background(), &pb.ClientDoneRequest{
			ClientId: "Producer",
		})
	}
}
