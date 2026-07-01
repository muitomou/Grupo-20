# Laboratorio 3: Sistemas Distribuidos - DistriEats

## Integrantes (Grupo 20)
* Mauro Castillo | Rol: 202273627-5
* Ignacio Casanova | Rol: 202273631-3
* Nicolás Ortíz | Rol: 202273528-7

## Instrucciones de Ejecución
Este lab está completamente aislado en contenedores Docker y orquestado con Make.
Para ejecutar la simulación, debe conectarse a cada una de las 4 máquinas virtuales (VMs) correspondientes y ejecutar su respectivo target localmente en la raíz del proyecto Grupo-20/. Orden recomendado:

1. **VM2 (dist078)**: `make docker-VM2` (Levanta Datanode 1, el Gateway Coordinador y Cliente 1)
2. **VM3 (dist079)**: `make docker-VM3` (Levanta Datanode 2 y Cliente 2)
3. **VM4 (dist080)**: `make docker-VM4` (Levanta Datanode 3 y Cliente 3)
4. **VM1 (dist077)**: `make docker-VM1` (Levanta el Broker Central e inicia el Productor con la simulación del CSV)

Opcionalmente se puede ejecutar `docker logs -f producer` **desde la VM1 (dist077)** para ver al Producer en tiempo real. Cuando este termine dira 'Simulación finalizada' y estará listo para generar la auditoría.
Para generar y revisar el reporte final de auditoría, ejecute `make auditoria` **desde la VM1 (dist077)**. El sistema asegurará convergencia estricta antes de emitir el documento.
Para limpiar los contenedores en cada máquina, ejecute `make clean`.

### Prueba de Tolerancia a Fallos (Fase 4)
Este ecosistema es resistente a caídas de nodos gracias al uso de reintentos y relojes vectoriales. Para poner a prueba la tolerancia a fallos durante la simulación:
1. Mientras el Productor inyecta eventos, diríjase a la VM2 (o VM3/VM4).
2. Detenga abruptamente un datanode con el comando: `docker stop datanode1`
3. Observará que el sistema no colapsa y los demás nodos continúan operando.
4. Vuelva a iniciarlo con: `docker start datanode1`
5. El protocolo Gossip (sincronización en background) automáticamente pondrá al día a este nodo con la historia que se perdió durante su caída, permitiendo converger correctamente antes del reporte final.
