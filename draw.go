package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"guardia/world"
)

// Paleta de colores.
var (
	colBG       = color.NRGBA{18, 19, 24, 255}
	colFloor    = color.NRGBA{38, 41, 52, 255}
	colWall     = color.NRGBA{105, 112, 130, 255}
	colCarpet   = color.NRGBA{98, 66, 52, 255}
	colVision   = color.NRGBA{255, 230, 80, 38}
	colPath     = color.NRGBA{255, 215, 60, 255}
	colGuard    = color.NRGBA{235, 70, 70, 255}
	colPlayer   = color.NRGBA{70, 220, 120, 255}
	colWaypoint = color.NRGBA{80, 90, 120, 255}
	colPanel    = color.NRGBA{12, 13, 17, 255}
)

// px es un píxel blanco de 1x1 que se estira y tiñe para dibujar rectángulos.
var px = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}()

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(px, &op)
}

// fillCell pinta una celda dejando 1 px de separación (se ve la cuadrícula).
func fillCell(dst *ebiten.Image, p world.Point, c color.Color) {
	fillRect(dst, float64(p.X*cellSize), float64(p.Y*cellSize), cellSize-1, cellSize-1, c)
}

// Draw dibuja un cuadro. Solo lee el estado; nunca lo modifica.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	grid := g.lv.Grid

	// 1. Terreno
	for y := 0; y < grid.H; y++ {
		for x := 0; x < grid.W; x++ {
			p := world.Point{X: x, Y: y}
			c := color.Color(colFloor)
			switch grid.At(p) {
			case world.Wall:
				c = colWall
			case world.Carpet:
				c = colCarpet
			}
			fillCell(screen, p, c)
		}
	}
	for _, p := range g.lv.Patrol { // puntos de patrulla
		fillRect(screen, float64(p.X*cellSize+9), float64(p.Y*cellSize+9), 5, 5, colWaypoint)
	}

	// 2. Cono de visión del guardia
	for y := 0; y < grid.H; y++ {
		for x := 0; x < grid.W; x++ {
			p := world.Point{X: x, Y: y}
			if grid.Walkable(p) && g.guard.Sees(grid, p) {
				fillCell(screen, p, colVision)
			}
		}
	}

	// 3. Nodos explorados por la búsqueda (más intensos = expandidos más tarde)
	if g.showExplored {
		n := len(g.cur.Visited)
		for i, p := range g.cur.Visited {
			a := uint8(35 + 110*i/max(n, 1))
			fillCell(screen, p, color.NRGBA{60, 170, 255, a})
		}
	}

	// 4. Camino planificado
	for _, p := range g.cur.Path {
		fillRect(screen, float64(p.X*cellSize+8), float64(p.Y*cellSize+8), 7, 7, colPath)
	}

	// 5. Actores
	gx, gy := g.guard.Pos()
	fillRect(screen, gx*cellSize+3, gy*cellSize+3, cellSize-7, cellSize-7, colGuard)
	fx := gx*cellSize + cellSize/2 - 2 + math.Cos(g.guard.Facing)*8
	fy := gy*cellSize + cellSize/2 - 2 + math.Sin(g.guard.Facing)*8
	fillRect(screen, fx, fy, 4, 4, color.White) // indicador de hacia dónde mira
	fillRect(screen, float64(g.player.X*cellSize+3), float64(g.player.Y*cellSize+3), cellSize-7, cellSize-7, colPlayer)

	// 6. Aviso temporal y panel lateral
	if g.msgTicks > 0 {
		fillRect(screen, 0, 0, float64(grid.W*cellSize), 20, color.NRGBA{0, 0, 0, 200})
		ebitenutil.DebugPrintAt(screen, g.msg, 8, 2)
	}
	g.drawHUD(screen)
}

// drawHUD escribe el panel lateral: estado, métricas del último plan, tabla
// comparativa y controles. (Texto sin acentos: la fuente de depuración es ASCII.)
func (g *Game) drawHUD(screen *ebiten.Image) {
	x0 := g.lv.Grid.W * cellSize
	fillRect(screen, float64(x0), 0, panelW, float64(g.lv.Grid.H*cellSize), colPanel)

	mode := "JUEGO"
	if g.analysis {
		mode = "ANALISIS"
	}
	yn := func(b bool) string {
		if b {
			return "SI"
		}
		return "NO"
	}
	lines := []string{
		"GUARDIA - Practica integradora U2",
		fmt.Sprintf("Mapa %d/%d: %s", g.li+1, len(world.Levels()), g.lv.Name),
		"Algoritmo: " + g.algo.String(),
		fmt.Sprintf("Diagonales: %s   Modo: %s", yn(g.diag), mode),
		fmt.Sprintf("Guardia: %s   Capturas: %d", g.guard.State, g.captures),
		"",
		"ULTIMO PLAN (" + g.cur.Algo.Short() + ")",
	}
	if g.cur.Found {
		lines = append(lines,
			fmt.Sprintf(" Pasos: %d   Costo: %.1f", g.cur.Steps(), g.cur.Cost),
			fmt.Sprintf(" Expandidos: %d  Generados: %d", g.cur.Expanded, g.cur.Generated),
			fmt.Sprintf(" Frontera max: %d", g.cur.MaxFrontier),
			fmt.Sprintf(" Tiempo: %.0f us", float64(g.cur.Duration.Nanoseconds())/1000))
	} else {
		lines = append(lines, " (aun sin ruta)", "", "", "")
	}
	lines = append(lines, "", "COMPARATIVA guardia -> jugador", "  Alg     Exp  Gen  Costo Pasos    us")
	for _, r := range g.cmp {
		mark := " "
		if r.Algo == g.algo {
			mark = ">"
		}
		if !r.Found {
			lines = append(lines, fmt.Sprintf("%s %-7s sin ruta", mark, r.Algo.Short()))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %-7s%4d %4d %6.1f %4d %5.0f", mark, r.Algo.Short(),
			r.Expanded, r.Generated, r.Cost, r.Steps(), float64(r.Duration.Nanoseconds())/1000))
	}
	lines = append(lines, "", "CONTROLES",
		" WASD/Flechas: mover jugador",
		" 1 BFS  2 UCS  3 Greedy  4 A*",
		" M: mapa      R: reiniciar",
		" V: diagonales  E: explorados",
		" X: modo analisis (clic izq=jugador,",
		"    clic der=guardia)",
		" F: guardar tabla en metricas.csv")

	for i, l := range lines {
		ebitenutil.DebugPrintAt(screen, l, x0+8, 6+i*16)
	}
}
