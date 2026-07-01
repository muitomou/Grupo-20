# Laboratorio 3: Sistemas Distribuidos - DistriEats

## Integrantes (Grupo 20)
* Mauro Castillo | Rol: 202273627-5
* Ignacio Casanova | Rol: 202273631-3
* Nicolás Ortíz | Rol: 202273528-7

## Instrucciones de Ejecución
Este ecosistema logístico está completamente aislado en contenedores Docker y orquestado con Make.
Para ejecutar la simulación, abra 4 terminales en el directorio raíz del proyecto y ejecute en orden secuencial:

1. `make docker-VM1` (Levanta el Broker Central e inicia la simulación del CSV)
2. `make docker-VM2` (Levanta Datanode 1, el Gateway Coordinador y Cliente 1)
3. `make docker-VM3` (Levanta Datanode 2 y Cliente 2)
4. `make docker-VM4` (Levanta Datanode 3 y Cliente 3)

Para generar el reporte final de auditoría ejecutar make report
Para limpiar los contenedores ejecutar make clean

