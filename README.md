# Guardia: estrategias de búsqueda (BFS / UCS / Greedy / A*)

Juego de sigilo 2D en Go + Ebitengine. Un guardia (agente basado en metas) busca
al jugador y la pantalla compara las cuatro estrategias en vivo.

## Cómo ejecutarlo

```bash
go mod tidy      # descarga Ebitengine (la primera vez)
go run .         # el juego
go run ./cmd/bench          # comparativa sin ventana (4 direcciones)
go run ./cmd/bench -diag    # comparativa con 8 direcciones
go test ./...    # pruebas de los algoritmos y del comportamiento del guardia
```

Ebitengine reciente exige una versión de Go reciente (`go mod tidy` te avisa).
En Linux hacen falta las librerías de sistema de Ebitengine:
`sudo apt install libc6-dev libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config`
(en Windows y macOS no hace falta nada extra).

## Controles

| Tecla | Acción |
|---|---|
| WASD / flechas | mover al jugador |
| 1 / 2 / 3 / 4 | BFS / UCS / Greedy / A* |
| M / R | siguiente mapa / reiniciar |
| V | diagonales sí/no |
| E | mostrar nodos explorados |
| X | modo análisis: el guardia no se mueve; clic izq = coloca al jugador, clic der = coloca al guardia |
| F | agrega la tabla comparativa a `metricas.csv` |

## Estructura (y a qué sección del reporte alimenta)

| Archivo | Contenido | Sección del índice |
|---|---|---|
| `world/grid.go` | cuadrícula, vecinos, costos, línea de vista | 3.5 espacio de estados |
| `world/maps.go` | los 4 mapas de prueba | 3.2, 6.2 |
| `search/search.go` | BFS, UCS, Greedy, A*, g(n), h(n), rastreo inverso, métricas | 4.1, 4.2, 5.1 |
| `agent.go` | guardia: sensores, máquina de estados, actuadores | 3.1 |
| `game.go`, `draw.go` | ciclo del juego, interfaz, HUD, exportar CSV | 3.4, 6.1 |
| `cmd/bench` | comparativa reproducible | 6.2 |
| `*_test.go` | pruebas | anexos |
