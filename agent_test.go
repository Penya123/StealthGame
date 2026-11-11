package main

import (
	"testing"

	"guardia/search"
	"guardia/world"
)

// Jugador "invisible": fuera del mapa, nunca entra en el cono de visión.
var hidden = world.Point{X: -50, Y: -50}

func plannerFor(lv world.Level, algo search.Algorithm) Planner {
	return func(from, goal world.Point) search.Result {
		return search.Run(lv.Grid, from, goal, true, algo)
	}
}

// El guardia sin ver a nadie debe recorrer todos sus puntos de patrulla.
func TestPatrullaVisitaTodosLosPuntos(t *testing.T) {
	for _, lv := range world.Levels() {
		for _, algo := range search.AllAlgorithms {
			gd := NewGuard(lv)
			plan := plannerFor(lv, algo)
			seen := map[world.Point]bool{}
			for tick := 0; tick < 20000; tick++ {
				gd.Update(lv.Grid, hidden, plan)
				seen[gd.Cell] = true
			}
			for _, wp := range lv.Patrol {
				if !seen[wp] {
					t.Errorf("%s/%v: nunca llegó al punto de patrulla %v", lv.Name, algo, wp)
				}
			}
		}
	}
}

// Ve al jugador -> persigue -> lo alcanza.
func TestPersecucionAlcanzaAlJugador(t *testing.T) {
	lv := world.Levels()[0] // "Abierto"
	gd := NewGuard(lv)
	player := world.Point{X: 8, Y: 2} // justo al este: Facing inicial 0 mira hacia allá
	plan := plannerFor(lv, search.AStar)

	sawChase := false
	for tick := 0; tick < 2000; tick++ {
		gd.Update(lv.Grid, player, plan)
		if gd.State == Chase {
			sawChase = true
		}
	}
	if !sawChase {
		t.Fatal("el guardia nunca pasó a PERSECUCION")
	}
	if gd.Cell != player {
		t.Fatalf("el guardia terminó en %v y no alcanzó al jugador en %v", gd.Cell, player)
	}
}

// Persecución -> pierde al jugador -> búsqueda -> vuelve a patrullar.
func TestPerderAlJugadorVuelveAPatrullar(t *testing.T) {
	lv := world.Levels()[0]
	gd := NewGuard(lv)
	plan := plannerFor(lv, search.AStar)
	player := world.Point{X: 8, Y: 2}

	for tick := 0; tick < 30 && gd.State != Chase; tick++ {
		gd.Update(lv.Grid, player, plan)
	}
	if gd.State != Chase {
		t.Fatal("no entró en persecución")
	}
	visited := map[State]bool{}
	for tick := 0; tick < 5000; tick++ {
		gd.Update(lv.Grid, hidden, plan) // el jugador desaparece
		visited[gd.State] = true
	}
	if !visited[Search] {
		t.Error("nunca pasó por BUSQUEDA")
	}
	if gd.State != Patrol {
		t.Errorf("terminó en %v; debía regresar a PATRULLA", gd.State)
	}
}

// Una pared bloquea la visión aunque el jugador esté dentro del alcance.
func TestMuroBloqueaVision(t *testing.T) {
	grid := world.NewGrid(10, 3)
	grid.Set(world.Point{X: 5, Y: 1}, world.Wall)
	gd := &Guard{Cell: world.Point{X: 1, Y: 1}}
	if gd.Sees(grid, world.Point{X: 8, Y: 1}) {
		t.Error("vio a través de un muro")
	}
	if !gd.Sees(grid, world.Point{X: 4, Y: 1}) {
		t.Error("no vio a un jugador despejado y cercano")
	}
	gd.Facing = 3.14159 // de espaldas
	if gd.Sees(grid, world.Point{X: 4, Y: 1}) {
		t.Error("vio de espaldas fuera del radio de oído")
	}
}
