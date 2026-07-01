IMAGE_NAME = distrieats-img
BROKER_HOST = dist077.inf.santiago.usm.cl
GATEWAY_HOST = dist078.inf.santiago.usm.cl

build:
	docker build -t $(IMAGE_NAME) .

docker-VM1: build
	docker run -d --name broker -p 9090:9090 $(IMAGE_NAME) broker --port 9090 --datanodes dist078.inf.santiago.usm.cl:50051,dist079.inf.santiago.usm.cl:50052,dist080.inf.santiago.usm.cl:50053 --expected 4
	sleep 2
	docker run -d --name producer $(IMAGE_NAME) producer --broker $(BROKER_HOST):9090 --csv pedidos.csv

docker-VM2: build
	docker run -d --name datanode1 -p 50051:50051 $(IMAGE_NAME) datanode --port 50051 --id DN1 --peers dist079.inf.santiago.usm.cl:50052,dist080.inf.santiago.usm.cl:50053
	sleep 2
	docker run -d --name gateway -p 8080:8080 $(IMAGE_NAME) gateway --port 8080 --broker $(BROKER_HOST):9090
	sleep 5
	docker run --rm --name cliente1 $(IMAGE_NAME) cliente --gateway $(GATEWAY_HOST):8080 --id Cliente-1

docker-VM3: build
	docker run -d --name datanode2 -p 50052:50052 $(IMAGE_NAME) datanode --port 50052 --id DN2 --peers dist078.inf.santiago.usm.cl:50051,dist080.inf.santiago.usm.cl:50053
	sleep 5
	docker run --rm --name cliente2 $(IMAGE_NAME) cliente --gateway $(GATEWAY_HOST):8080 --id Cliente-2

docker-VM4: build
	docker run -d --name datanode3 -p 50053:50053 $(IMAGE_NAME) datanode --port 50053 --id DN3 --peers dist078.inf.santiago.usm.cl:50051,dist079.inf.santiago.usm.cl:50052
	sleep 5
	docker run --rm --name cliente3 $(IMAGE_NAME) cliente --gateway $(GATEWAY_HOST):8080 --id Cliente-3

clean:
	docker stop broker producer datanode1 datanode2 datanode3 gateway || true
	docker rm broker producer datanode1 datanode2 datanode3 gateway || true
	rm -f Reporte.txt

auditoria:
	@echo "Esperando a que el Broker (en VM1) termine de emitir el reporte final..."
	@sleep 20
	@if [ -f Reporte.txt ]; then cat Reporte.txt; else echo "Reporte.txt no encontrado aún. Revisa logs del broker."; fi