// Guardia: juego de sigilo 2D donde un guardia (agente basado en metas) busca al
// jugador con BFS, UCS, Greedy o A*, y la pantalla compara las estrategias en vivo.
//
// Ejecutar:
//
//	go mod tidy
//	go run .
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	g := NewGame()
	w, h := g.Layout(0, 0)
	ebiten.SetWindowSize(w*5/4, h*5/4)
	ebiten.SetWindowTitle("Guardia - Estrategias de busqueda (BFS / UCS / Greedy / A*)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
