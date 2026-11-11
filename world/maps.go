package world

import (
	"fmt"
	"sort"
)

// Level es un escenario de prueba: el mapa, dónde empiezan el guardia y el
// jugador, y los puntos por los que patrulla el guardia.
type Level struct {
	Name        string
	Description string
	Grid        *Grid
	GuardStart  Point
	PlayerStart Point
	Patrol      []Point // puntos '1'..'9' del mapa, en orden
}

// Parse construye un Level a partir de un dibujo en ASCII:
//
//	'#' muro   '.' piso   ',' alfombra
//	'G' guardia   'P' jugador   '1'..'9' puntos de patrulla (sobre piso)
//
// Entra en pánico si el dibujo no es rectangular: es un error de programación.
func Parse(name, desc string, rows []string) Level {
	h := len(rows)
	w := len(rows[0])
	lv := Level{Name: name, Description: desc, Grid: NewGrid(w, h)}
	type wp struct {
		n int
		p Point
	}
	var wps []wp
	for y, row := range rows {
		if len(row) != w {
			panic(fmt.Sprintf("mapa %q: la fila %d mide %d y debería medir %d", name, y, len(row), w))
		}
		for x, ch := range row {
			p := Point{x, y}
			switch {
			case ch == '#':
				lv.Grid.Set(p, Wall)
			case ch == ',':
				lv.Grid.Set(p, Carpet)
			case ch == 'G':
				lv.GuardStart = p
			case ch == 'P':
				lv.PlayerStart = p
			case ch >= '1' && ch <= '9':
				wps = append(wps, wp{int(ch - '0'), p})
			}
		}
	}
	sort.Slice(wps, func(i, j int) bool { return wps[i].n < wps[j].n })
	for _, w := range wps {
		lv.Patrol = append(lv.Patrol, w.p)
	}
	return lv
}

// Levels devuelve una copia nueva de todos los escenarios.
// Cada uno está pensado para resaltar una diferencia entre los algoritmos.
func Levels() []Level {
	defs := []struct {
		name, desc string
		rows       []string
	}{
		{
			"Abierto",
			"Casi sin obstaculos: A* y Greedy se parecen, BFS expande de mas.",
			[]string{
				"##############################",
				"#............................#",
				"#.G..........................#",
				"#...1....................2...#",
				"#.......##............##.....#",
				"#.......##............##.....#",
				"#............................#",
				"#............................#",
				"#............................#",
				"#..............##............#",
				"#..............##............#",
				"#............................#",
				"#............................#",
				"#............................#",
				"#.......##............##.....#",
				"#.......##............##.....#",
				"#...4....................3...#",
				"#..........................P.#",
				"#............................#",
				"##############################",
			},
		},
		{
			"Oficina",
			"Habitaciones con puertas: la ruta exige pasar por varias aberturas.",
			[]string{
				"##############################",
				"#.........#.........#........#",
				"#.G.......#.........#........#",
				"#..............2....#........#",
				"#.........#.........#........#",
				"#.........#.........#....3...#",
				"#....1....#######...#........#",
				"#.........#.........#........#",
				"#.........#.........#........#",
				"#####.#####..........####.####",
				"#.........#.........#........#",
				"#.........#.........#........#",
				"#.........#.........#........#",
				"#.........#...#######........#",
				"#.........#.........#........#",
				"#...................#........#",
				"#.........#....4.............#",
				"#.........#.........#......P.#",
				"#.........#.........#........#",
				"##############################",
			},
		},
		{
			"Trampa en U",
			"Una U abierta hacia el guardia: la heuristica lo atrae al fondo y debe rodear. Con diagonales Greedy deja de ser optimo.",
			[]string{
				"##############################",
				"#............................#",
				"#............................#",
				"#..1......................2..#",
				"#............................#",
				"#..........########..........#",
				"#.................#..........#",
				"#.................#..........#",
				"#.................#..........#",
				"#.................#..........#",
				"#..G..............#.P........#",
				"#.................#..........#",
				"#.................#..........#",
				"#.................#..........#",
				"#..........########..........#",
				"#............................#",
				"#..4......................3..#",
				"#............................#",
				"#............................#",
				"##############################",
			},
		},
		{
			"Alfombra",
			"Zona de alfombra (costo 3): BFS cruza por el centro, A* y UCS rodean.",
			[]string{
				"##############################",
				"#............................#",
				"#..1......................2..#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,##,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.G.....,,,,,,,,,,,,,,.....P.#",
				"#.......,,,,,##,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#.......,,,,,,,,,,,,,,.......#",
				"#..4......................3..#",
				"#............................#",
				"##############################",
			},
		},
	}
	out := make([]Level, len(defs))
	for i, d := range defs {
		out[i] = Parse(d.name, d.desc, d.rows)
	}
	return out
}
