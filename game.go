package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"guardia/search"
	"guardia/world"
)

const (
	cellSize = 24  // píxeles por celda
	panelW   = 300 // ancho del panel lateral (HUD)
)

// cmpKey y planKey sirven para recalcular solo cuando algo cambió.
type cmpKey struct {
	from, to world.Point
	diag     bool
	level    int
}

type planKey struct {
	from, to world.Point
	algo     search.Algorithm
	diag     bool
}

// Game contiene todo el estado del juego y cumple la interfaz ebiten.Game.
type Game struct {
	li     int // índice del mapa actual
	lv     world.Level
	player world.Point
	cool   int // ticks que faltan para que el jugador pueda moverse de nuevo
	guard  *Guard

	algo         search.Algorithm // algoritmo que usa el guardia para planificar
	diag         bool             // ¿movimiento en 8 direcciones?
	showExplored bool             // dibujar los nodos expandidos
	analysis     bool             // modo análisis: el guardia no se mueve, solo planifica

	cur       search.Result // último plan (se dibuja y aparece en el HUD)
	cmp       []search.Result
	cmpKey    cmpKey
	cmpValid  bool
	planKey   planKey
	planValid bool

	captures int
	msg      string
	msgTicks int
}

// NewGame crea el juego en el primer mapa con A* y diagonales activadas.
func NewGame() *Game {
	g := &Game{algo: search.AStar, diag: true, showExplored: true}
	g.loadLevel(0)
	return g
}

func (g *Game) loadLevel(i int) {
	levels := world.Levels()
	g.li = i % len(levels)
	g.lv = levels[g.li]
	g.player = g.lv.PlayerStart
	g.guard = NewGuard(g.lv)
	g.cur = search.Result{}
	g.cool = 0
	g.cmpValid, g.planValid = false, false
}

// plan es el Planner que se le entrega al guardia.
func (g *Game) plan(from, goal world.Point) search.Result {
	return search.Run(g.lv.Grid, from, goal, g.diag, g.algo)
}

func (g *Game) flash(s string) { g.msg, g.msgTicks = s, 120 }

// Update se llama 60 veces por segundo.
func (g *Game) Update() error {
	g.handleKeys()
	g.movePlayer()

	if g.analysis {
		g.analyze()
	} else {
		if res, planned := g.guard.Update(g.lv.Grid, g.player, g.plan); planned {
			g.cur = res
		}
		g.checkCapture()
	}
	g.refreshComparison()

	if g.msgTicks > 0 {
		g.msgTicks--
	}
	return nil
}

func (g *Game) handleKeys() {
	algoKeys := []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4}
	for i, k := range algoKeys {
		if inpututil.IsKeyJustPressed(k) {
			g.algo = search.AllAlgorithms[i]
		}
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyM):
		g.loadLevel(g.li + 1)
	case inpututil.IsKeyJustPressed(ebiten.KeyR):
		g.loadLevel(g.li)
	case inpututil.IsKeyJustPressed(ebiten.KeyV):
		g.diag = !g.diag
	case inpututil.IsKeyJustPressed(ebiten.KeyE):
		g.showExplored = !g.showExplored
	case inpututil.IsKeyJustPressed(ebiten.KeyX):
		g.analysis = !g.analysis
		g.guard.replan = true
		g.planValid = false
	case inpututil.IsKeyJustPressed(ebiten.KeyF):
		g.exportCSV()
	}

	// En modo análisis se puede colocar a los actores con el ratón.
	if g.analysis {
		mx, my := ebiten.CursorPosition()
		cell := world.Point{X: mx / cellSize, Y: my / cellSize}
		if g.lv.Grid.Walkable(cell) {
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				g.player = cell
			}
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
				g.guard.Cell, g.guard.Path, g.guard.Progress = cell, nil, 0
			}
		}
	}
}

// movePlayer mueve al jugador celda por celda. Cruzar alfombra es más lento.
func (g *Game) movePlayer() {
	if g.cool > 0 {
		g.cool--
		return
	}
	var d world.Point
	switch {
	case ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW):
		d = world.Point{Y: -1}
	case ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS):
		d = world.Point{Y: 1}
	case ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA):
		d = world.Point{X: -1}
	case ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD):
		d = world.Point{X: 1}
	default:
		return
	}
	np := world.Point{X: g.player.X + d.X, Y: g.player.Y + d.Y}
	if !g.lv.Grid.Walkable(np) {
		return
	}
	g.player = np
	base := 9.0
	if g.analysis {
		base = 4 // más ágil para posicionarse al tomar capturas
	}
	g.cool = int(base * g.lv.Grid.At(np).Cost())
}

// checkCapture reinicia el nivel si el guardia alcanzó al jugador.
func (g *Game) checkCapture() {
	gx, gy := g.guard.Pos()
	close := math.Hypot(gx-float64(g.player.X), gy-float64(g.player.Y)) < 0.9
	if close && g.lv.Grid.LineOfSight(g.guard.Cell, g.player) {
		c := g.captures + 1
		g.loadLevel(g.li)
		g.captures = c
		g.flash("TE ATRAPARON")
	}
}

// analyze (modo análisis): replanifica solo cuando cambia algo de la escena.
func (g *Game) analyze() {
	k := planKey{g.guard.Cell, g.player, g.algo, g.diag}
	if g.planValid && k == g.planKey {
		return
	}
	g.planKey, g.planValid = k, true
	g.cur = g.plan(g.guard.Cell, g.player)
}

// refreshComparison corre los 4 algoritmos sobre la MISMA situación (guardia ->
// jugador) para mostrar la tabla comparativa en vivo.
func (g *Game) refreshComparison() {
	k := cmpKey{g.guard.Cell, g.player, g.diag, g.li}
	if g.cmpValid && k == g.cmpKey {
		return
	}
	g.cmpKey, g.cmpValid = k, true
	g.cmp = g.cmp[:0]
	for _, a := range search.AllAlgorithms {
		g.cmp = append(g.cmp, search.RunAvg(g.lv.Grid, k.from, k.to, g.diag, a, 5))
	}
}

// exportCSV agrega la comparativa actual a metricas.csv (evidencia para el reporte).
func (g *Game) exportCSV() {
	const name = "metricas.csv"
	_, err := os.Stat(name)
	isNew := os.IsNotExist(err)
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		g.flash("No se pudo escribir " + name)
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if isNew {
		w.Write([]string{"mapa", "algoritmo", "diagonales", "origen", "destino", "encontrado",
			"pasos", "costo", "expandidos", "generados", "frontera_max", "tiempo_us"})
	}
	for _, r := range g.cmp {
		w.Write([]string{g.lv.Name, r.Algo.Short(), fmt.Sprint(g.diag),
			fmt.Sprintf("(%d,%d)", g.cmpKey.from.X, g.cmpKey.from.Y),
			fmt.Sprintf("(%d,%d)", g.cmpKey.to.X, g.cmpKey.to.Y),
			fmt.Sprint(r.Found), fmt.Sprint(r.Steps()), fmt.Sprintf("%.2f", r.Cost),
			fmt.Sprint(r.Expanded), fmt.Sprint(r.Generated), fmt.Sprint(r.MaxFrontier),
			fmt.Sprintf("%.1f", float64(r.Duration.Nanoseconds())/1000)})
	}
	w.Flush()
	g.flash("Metricas agregadas a " + name)
}

// Layout fija la resolución lógica (se escala al tamaño de la ventana).
func (g *Game) Layout(_, _ int) (int, int) {
	return g.lv.Grid.W*cellSize + panelW, g.lv.Grid.H * cellSize
}
