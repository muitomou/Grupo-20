IMAGE_NAME = distrieats-img
BROKER_HOST = dist077.inf.santiago.usm.cl
GATEWAY_HOST = dist078.inf.santiago.usm.cl

build:
	docker build -t $(IMAGE_NAME) .

# VM1: Broker y Productor
docker-VM1: build
	docker run -d --name broker -p 9090:9090 $(IMAGE_NAME) broker --port 9090 --datanodes dist078.inf.santiago.usm.cl:50051,dist079.inf.santiago.usm.cl:50052,dist080.inf.santiago.usm.cl:50053
	sleep 2
	docker run -d --name producer $(IMAGE_NAME) producer --broker $(BROKER_HOST):9090 --csv pedidos.csv

# VM2: Datanode 1, Gateway y Cliente 1
docker-VM2: build
	docker run -d --name datanode1 -p 50051:50051 $(IMAGE_NAME) datanode --port 50051 --id DN1 --peers dist079.inf.santiago.usm.cl:50052,dist080.inf.santiago.usm.cl:50053
	sleep 2
	docker run -d --name gateway -p 8080:8080 $(IMAGE_NAME) gateway --port 8080 --broker $(BROKER_HOST):9090
	sleep 5
	docker run --rm --name cliente1 $(IMAGE_NAME) cliente --gateway $(GATEWAY_HOST):8080 --id Cliente-1 > logs_cliente1.txt 2>&1

# VM3: Datanode 2 y Cliente 2
docker-VM3: build
	docker run -d --name datanode2 -p 50052:50052 $(IMAGE_NAME) datanode --port 50052 --id DN2 --peers dist078.inf.santiago.usm.cl:50051,dist080.inf.santiago.usm.cl:50053
	sleep 5
	docker run --rm --name cliente2 $(IMAGE_NAME) cliente --gateway $(GATEWAY_HOST):8080 --id Cliente-2 > logs_cliente2.txt 2>&1

# VM4: Datanode 3 y Cliente 3
docker-VM4: build
	docker run -d --name datanode3 -p 50053:50053 $(IMAGE_NAME) datanode --port 50053 --id DN3 --peers dist078.inf.santiago.usm.cl:50051,dist079.inf.santiago.usm.cl:50052
	sleep 5
	docker run --rm --name cliente3 $(IMAGE_NAME) cliente --gateway $(GATEWAY_HOST):8080 --id Cliente-3 > logs_cliente3.txt 2>&1

clean:
	docker stop broker producer datanode1 datanode2 datanode3 gateway || true
	docker rm broker producer datanode1 datanode2 datanode3 gateway || true
	rm -f logs_*.txt Reporte.txt

collect-logs:
	@echo "--- LOGS LOCALES DE $(shell hostname) ---" > logs_finales.txt
	@docker logs broker 2>/dev/null >> logs_finales.txt || true
	@docker logs datanode1 2>/dev/null >> logs_finales.txt || true
	@docker logs datanode2 2>/dev/null >> logs_finales.txt || true
	@docker logs datanode3 2>/dev/null >> logs_finales.txt || true
	@docker logs gateway 2>/dev/null >> logs_finales.txt || true
	@cat logs_cliente*.txt 2>/dev/null >> logs_finales.txt || true
	@echo "Archivo logs_finales.txt generado."

auditoria:
	@echo "Esperando 15 segundos para convergencia..."
	@sleep 15
	@echo "Generando estados finales..."
	@# Aquí debes llamar a un comando que cada datanode tenga para exportar su log
	@docker exec datanode1 sh -c "cat /app/logs_finales.txt" > log_dn1.txt
	@docker exec datanode2 sh -c "cat /app/logs_finales.txt" > log_dn2.txt
	@docker exec datanode3 sh -c "cat /app/logs_finales.txt" > log_dn3.txt
	@echo "Verificando consistencia..."
	@diff log_dn1.txt log_dn2.txt > /dev/null && diff log_dn2.txt log_dn3.txt > /dev/null && echo "CONVERGENCIA EXITOSA: Los nodos son idénticos" || echo "ERROR: Los estados no coinciden"
	@cat log_dn1.txt log_dn2.txt log_dn3.txt > Reporte.txt