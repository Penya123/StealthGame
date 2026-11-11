package search

import (
	"math"
	"testing"

	"guardia/world"
)

// Todos los caminos deben ser válidos: empiezan y terminan donde deben y cada
// paso va a una celda vecina transitable.
func validPath(t *testing.T, g *world.Grid, r Result, start, goal world.Point, diag bool) {
	t.Helper()
	if !r.Found {
		t.Fatalf("%v no encontró camino", r.Algo)
	}
	if r.Path[0] != start || r.Path[len(r.Path)-1] != goal {
		t.Fatalf("%v: camino no une inicio y meta", r.Algo)
	}
	for i := 1; i < len(r.Path); i++ {
		ok := false
		for _, st := range g.Neighbors(r.Path[i-1], diag) {
			if st.To == r.Path[i] {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("%v: paso inválido %v -> %v", r.Algo, r.Path[i-1], r.Path[i])
		}
	}
}

func TestOptimalidad(t *testing.T) {
	for _, lv := range world.Levels() {
		for _, diag := range []bool{false, true} {
			res := map[Algorithm]Result{}
			for _, a := range AllAlgorithms {
				res[a] = Run(lv.Grid, lv.GuardStart, lv.PlayerStart, diag, a)
				validPath(t, lv.Grid, res[a], lv.GuardStart, lv.PlayerStart, diag)
			}
			// A* y UCS son óptimos en costo: deben coincidir.
			if math.Abs(res[AStar].Cost-res[UCS].Cost) > 1e-6 {
				t.Errorf("%s diag=%v: A* %.3f != UCS %.3f", lv.Name, diag, res[AStar].Cost, res[UCS].Cost)
			}
			// Ningún algoritmo puede mejorar el costo óptimo.
			for _, a := range AllAlgorithms {
				if res[a].Cost < res[UCS].Cost-1e-6 {
					t.Errorf("%s: %v tiene costo menor que el óptimo", lv.Name, a)
				}
			}
			// BFS minimiza el número de pasos.
			for _, a := range AllAlgorithms {
				if res[a].Steps() < res[BFS].Steps() {
					t.Errorf("%s diag=%v: %v tiene menos pasos que BFS", lv.Name, diag, a)
				}
			}
			// A* con buena heurística nunca expande más que UCS.
			if res[AStar].Expanded > res[UCS].Expanded {
				t.Errorf("%s diag=%v: A* expandió más que UCS", lv.Name, diag)
			}
		}
	}
}

func TestSinCamino(t *testing.T) {
	g := world.NewGrid(5, 5)
	for y := 0; y < 5; y++ {
		g.Set(world.Point{X: 2, Y: y}, world.Wall) // muro que parte el mapa
	}
	for _, a := range AllAlgorithms {
		r := Run(g, world.Point{X: 0, Y: 0}, world.Point{X: 4, Y: 4}, false, a)
		if r.Found {
			t.Errorf("%v encontró camino donde no hay", a)
		}
	}
}

func TestInicioIgualMeta(t *testing.T) {
	g := world.NewGrid(3, 3)
	p := world.Point{X: 1, Y: 1}
	for _, a := range AllAlgorithms {
		r := Run(g, p, p, true, a)
		if !r.Found || r.Steps() != 0 || r.Cost != 0 {
			t.Errorf("%v: caso inicio==meta incorrecto: %+v", a, r)
		}
	}
}
