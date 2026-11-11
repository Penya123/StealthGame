package main

import (
	"math"

	"guardia/search"
	"guardia/world"
)

// State es el comportamiento actual del guardia (máquina de estados).
type State int

const (
	Patrol State = iota // recorre sus puntos de patrulla
	Chase               // ve al jugador: persigue su posición actual
	Search              // lo perdió de vista: va a la última posición vista
)

func (s State) String() string {
	switch s {
	case Patrol:
		return "PATRULLA"
	case Chase:
		return "PERSECUCION"
	}
	return "BUSQUEDA"
}

// Parámetros de percepción y movimiento del guardia.
const (
	viewRange  = 9.0                 // alcance de la vista, en celdas
	viewFOV    = 100 * math.Pi / 180 // ángulo total del cono de visión
	hearRange  = 2.0                 // dentro de este radio "escucha" aunque esté de espaldas
	searchWait = 150                 // ticks que espera mirando alrededor al llegar a la última posición vista
)

// Planner calcula una ruta de from a goal con el algoritmo de búsqueda activo.
// Es la ÚNICA pieza que el agente necesita del módulo de búsqueda: así el
// guardia no sabe (ni le importa) si por dentro corre BFS, UCS, Greedy o A*.
type Planner func(from, goal world.Point) search.Result

// Guard es el agente basado en metas.
//
//	Sensores:   su posición y el cono de visión con línea de vista (Sees).
//	Actuadores: avanzar a una celda vecina y cambiar de estado.
//	Meta:       depende del estado (punto de patrulla, jugador o última posición vista).
type Guard struct {
	Cell     world.Point   // última celda alcanzada (posición lógica)
	Path     []world.Point // celdas pendientes; Path[0] es hacia donde camina
	Progress float64       // avance 0..1 entre Cell y Path[0] (para animar suave)
	Facing   float64       // hacia dónde mira, en radianes
	State    State
	Target   world.Point // meta de la ruta actual
	LastSeen world.Point // última celda donde vio al jugador

	patrol    []world.Point
	patrolIdx int
	wait      int
	replan    bool // fuerza recalcular al llegar al centro de la siguiente celda
}

// NewGuard crea un guardia en su posición inicial con la ruta de patrulla del nivel.
func NewGuard(lv world.Level) *Guard {
	return &Guard{Cell: lv.GuardStart, Target: lv.GuardStart, patrol: lv.Patrol, replan: true}
}

// Sees indica si el guardia ve la celda p: está dentro del alcance, dentro del
// cono de visión (o muy cerca) y sin ningún muro en medio.
func (gd *Guard) Sees(grid *world.Grid, p world.Point) bool {
	dx, dy := float64(p.X-gd.Cell.X), float64(p.Y-gd.Cell.Y)
	d := math.Hypot(dx, dy)
	if d > viewRange {
		return false
	}
	if d > hearRange {
		diff := math.Remainder(math.Atan2(dy, dx)-gd.Facing, 2*math.Pi)
		if math.Abs(diff) > viewFOV/2 {
			return false
		}
	}
	return grid.LineOfSight(gd.Cell, p)
}

// Pos devuelve la posición suavizada (en unidades de celda) para dibujarlo.
func (gd *Guard) Pos() (x, y float64) {
	x, y = float64(gd.Cell.X), float64(gd.Cell.Y)
	if len(gd.Path) > 0 {
		n := gd.Path[0]
		x += (float64(n.X) - x) * gd.Progress
		y += (float64(n.Y) - y) * gd.Progress
	}
	return
}

func (gd *Guard) setState(s State) {
	gd.State = s
	gd.replan = true
	gd.wait = 0
}

// Update ejecuta un tick (1/60 s) del ciclo percepción -> decisión -> acción.
// Devuelve el resultado de la búsqueda si en este tick se planificó una ruta.
func (gd *Guard) Update(grid *world.Grid, player world.Point, plan Planner) (res search.Result, planned bool) {
	// 1. PERCEPCIÓN
	sees := gd.Sees(grid, player)
	if sees {
		gd.LastSeen = player
	}

	// 2. TRANSICIONES DE ESTADO
	switch gd.State {
	case Patrol:
		if sees {
			gd.setState(Chase)
		}
	case Chase:
		if !sees {
			gd.setState(Search)
		}
	case Search:
		if sees {
			gd.setState(Chase)
		}
	}

	// 3. DECISIÓN: solo se decide (y se planifica) al estar justo en el centro de
	// una celda. Así el guardia nunca se "teletransporta" al recalcular.
	if gd.Progress == 0 {
		res, planned = gd.decide(plan)
	}

	// 4. ACCIÓN
	gd.advance(grid)
	return
}

// decide elige la meta según el estado y, si hace falta, pide una ruta.
func (gd *Guard) decide(plan Planner) (res search.Result, planned bool) {
	doPlan := func(goal world.Point) {
		res = plan(gd.Cell, goal)
		planned = true
		gd.Target = goal
		gd.replan = false
		gd.Path = nil
		if res.Found && len(res.Path) > 1 {
			gd.Path = res.Path[1:] // se quita la celda actual
		}
	}

	switch gd.State {
	case Chase:
		if gd.Cell == gd.LastSeen {
			return
		}
		// Recalcula cuando el jugador se movió (la meta cambió) o no hay ruta.
		if gd.replan || len(gd.Path) == 0 || gd.Target != gd.LastSeen {
			doPlan(gd.LastSeen)
		}

	case Search:
		if gd.Cell != gd.LastSeen {
			if gd.replan || len(gd.Path) == 0 {
				doPlan(gd.LastSeen)
			}
			return
		}
		// Llegó a donde lo vio por última vez: mira alrededor y luego vuelve a patrullar.
		gd.Facing += 0.06
		gd.wait++
		if gd.wait > searchWait {
			gd.setState(Patrol)
		}

	case Patrol:
		if len(gd.Path) == 0 && len(gd.patrol) > 0 {
			for i := 0; i < len(gd.patrol); i++ {
				t := gd.patrol[gd.patrolIdx]
				gd.patrolIdx = (gd.patrolIdx + 1) % len(gd.patrol)
				if t != gd.Cell {
					doPlan(t)
					break
				}
			}
		}
	}
	return
}

// advance camina por la ruta. Tardar más en terrenos caros (alfombra) o en
// diagonal hace que el costo g(n) de la búsqueda coincida con el TIEMPO real.
func (gd *Guard) advance(grid *world.Grid) {
	if len(gd.Path) == 0 {
		gd.Progress = 0
		return
	}
	next := gd.Path[0]
	gd.Facing = math.Atan2(float64(next.Y-gd.Cell.Y), float64(next.X-gd.Cell.X))
	base := 14.0 // ticks por celda de piso al patrullar
	if gd.State == Chase {
		base = 10 // persiguiendo corre más rápido
	}
	gd.Progress += 1 / (base * grid.StepCost(gd.Cell, next))
	if gd.Progress >= 1 {
		gd.Cell = next
		gd.Path = gd.Path[1:]
		gd.Progress = 0
	}
}
