# Laboratorio 3: Sistemas Distribuidos - DistriEats

## Integrantes (Grupo 20)
* Mauro Castillo | Rol: 202273627-5
* Ignacio Casanova | Rol: 202273631-3
* Nicolás Ortíz | Rol: 202273528-7

## Instrucciones de Ejecución
Este ecosistema logístico está completamente aislado en contenedores Docker y orquestado con Make.
Para ejecutar la simulación, debe conectarse a cada una de las 4 máquinas virtuales (VMs) correspondientes y ejecutar su respectivo target localmente en la raíz del proyecto. Le recomendamos iniciar los datanodes primero para que el broker se conecte sin problemas:

1. **VM2 (dist078)**: `make docker-VM2` (Levanta Datanode 1, el Gateway Coordinador y Cliente 1)
2. **VM3 (dist079)**: `make docker-VM3` (Levanta Datanode 2 y Cliente 2)
3. **VM4 (dist080)**: `make docker-VM4` (Levanta Datanode 3 y Cliente 3)
4. **VM1 (dist077)**: `make docker-VM1` (Levanta el Broker Central e inicia el Productor con la simulación del CSV)

Para generar y revisar el reporte final de auditoría, ejecute `make auditoria` **desde la VM1 (dist077)**. El sistema asegurará convergencia estricta antes de emitir el documento.
Para limpiar los contenedores en cada máquina, ejecute `make clean`.
