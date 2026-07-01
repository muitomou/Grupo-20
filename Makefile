IMAGE_NAME = distrieats-img
NETWORK_NAME = distrieats-net

build:
	docker build -t $(IMAGE_NAME) .
	docker network create $(NETWORK_NAME) || true

docker-VM1: build
	docker run -d --name broker --net $(NETWORK_NAME) -p 9090:9090 $(IMAGE_NAME) broker --port 9090 --datanodes datanode1:50051,datanode2:50052,datanode3:50053
	sleep 2
	docker run -d --name producer --net $(NETWORK_NAME) $(IMAGE_NAME) producer --broker broker:9090 --csv pedidos.csv

docker-VM2: build
	docker run -d --name datanode1 --net $(NETWORK_NAME) -p 50051:50051 $(IMAGE_NAME) datanode --port 50051 --id DN1 --peers datanode2:50052,datanode3:50053
	sleep 2
	docker run -d --name gateway --net $(NETWORK_NAME) -p 8080:8080 $(IMAGE_NAME) gateway --port 8080 --broker broker:9090
	sleep 5 
	docker run --rm --name cliente1 --net $(NETWORK_NAME) $(IMAGE_NAME) cliente --gateway gateway:8080 --id Cliente-1 > logs_cliente1.txt 2>&1

docker-VM3: build
	docker run -d --name datanode2 --net $(NETWORK_NAME) -p 50052:50052 $(IMAGE_NAME) datanode --port 50052 --id DN2 --peers datanode1:50051,datanode3:50053
	sleep 5
	docker run --rm --name cliente2 --net $(NETWORK_NAME) $(IMAGE_NAME) cliente --gateway gateway:8080 --id Cliente-2 > logs_cliente2.txt

docker-VM4: build
	docker run -d --name datanode3 --net $(NETWORK_NAME) -p 50053:50053 $(IMAGE_NAME) datanode --port 50053 --id DN3 --peers datanode1:50051,datanode2:50052
	sleep 10
	docker run --rm --name cliente3 --net $(NETWORK_NAME) $(IMAGE_NAME) cliente --gateway gateway:8080 --id Cliente-3 > logs_cliente3.txt

clean:
	docker stop broker producer datanode1 datanode2 datanode3 gateway || true
	docker rm broker producer datanode1 datanode2 datanode3 gateway || true
	docker network rm $(NETWORK_NAME) || true
	rm -f logs_*.txt Reporte.txt

report:
	@echo "=== REPORTE FINAL: DISTRIEATS ===" > Reporte.txt
	printf "\n[AUDITORIA READ YOUR WRITES]\n" >> Reporte.txt
	@cat logs_cliente*.txt >> Reporte.txt 2>/dev/null || echo "Logs de clientes no encontrados." >> Reporte.txt
	printf "\n[ESTADO GLOBAL DE PEDIDOS - Convergencia Alcanzada]\n" >> Reporte.txt
	@echo "--- DN1 ---" >> Reporte.txt
	@docker logs datanode1 2>&1 | grep "actualizado" >> Reporte.txt
	@echo "--- DN2 ---" >> Reporte.txt
	@docker logs datanode2 2>&1 | grep "actualizado" >> Reporte.txt
	@echo "--- DN3 ---" >> Reporte.txt
	@docker logs datanode3 2>&1 | grep "actualizado" >> Reporte.txt
	@echo "Reporte generado exitosamente: Reporte.txt"	


run-all: build
	docker run -d --name broker --net $(NETWORK_NAME) -p 9090:9090 $(IMAGE_NAME) broker --port 9090 --datanodes datanode1:50051,datanode2:50052,datanode3:50053
	docker run -d --name producer --net $(NETWORK_NAME) $(IMAGE_NAME) producer --broker broker:9090 --csv pedidos.csv
	docker run -d --name datanode1 --net $(NETWORK_NAME) -p 50051:50051 $(IMAGE_NAME) datanode --port 50051 --id DN1 --peers datanode2:50052,datanode3:50053
	docker run -d --name datanode2 --net $(NETWORK_NAME) -p 50052:50052 $(IMAGE_NAME) datanode --port 50052 --id DN2 --peers datanode1:50051,datanode3:50053
	docker run -d --name datanode3 --net $(NETWORK_NAME) -p 50053:50053 $(IMAGE_NAME) datanode --port 50053 --id DN3 --peers datanode1:50051,datanode2:50052
	docker run -d --name gateway --net $(NETWORK_NAME) -p 8080:8080 $(IMAGE_NAME) gateway --port 8080 --broker broker:9090
	sleep 10
	docker run --rm --name cliente1 --net $(NETWORK_NAME) $(IMAGE_NAME) cliente --gateway gateway:8080 --id Cliente-1 > logs_cliente1.txt 2>&1
	docker run --rm --name cliente2 --net $(NETWORK_NAME) $(IMAGE_NAME) cliente --gateway gateway:8080 --id Cliente-2 > logs_cliente2.txt 2>&1
	docker run --rm --name cliente3 --net $(NETWORK_NAME) $(IMAGE_NAME) cliente --gateway gateway:8080 --id Cliente-3 > logs_cliente3.txt 2>&1
	@$(MAKE) report


