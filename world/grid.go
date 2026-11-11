// Paquete world modela el entorno del juego como una cuadrícula (matriz de celdas).
//
// Esta misma estructura es, al mismo tiempo:
//   - el mapa que se dibuja en pantalla,
//   - el espacio de estados sobre el que buscan BFS, UCS, Greedy y A*,
//   - el "mundo" que percibe el guardia (línea de vista).
package world

import "math"

// Point es una celda (X = columna, Y = fila). Es comparable, así que se puede
// usar directamente como llave de un map (tabla hash).
type Point struct{ X, Y int }

// Cell es el tipo de terreno de una celda.
type Cell uint8

const (
	Floor  Cell = iota // piso normal, costo 1
	Wall               // muro, intransitable
	Carpet             // alfombra, costo 3 (terreno "caro")
)

// Cost devuelve el costo de ENTRAR a una celda de este tipo.
// Los muros devuelven +Inf porque no se puede entrar a ellos.
func (c Cell) Cost() float64 {
	switch c {
	case Floor:
		return 1
	case Carpet:
		return 3
	}
	return math.Inf(1)
}

// Grid es el mapa: W columnas por H filas, guardado en un arreglo plano.
type Grid struct {
	W, H  int
	Cells []Cell
}

// NewGrid crea una cuadrícula de w x h llena de piso.
func NewGrid(w, h int) *Grid {
	return &Grid{W: w, H: h, Cells: make([]Cell, w*h)}
}

// InBounds indica si p cae dentro del mapa.
func (g *Grid) InBounds(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < g.W && p.Y < g.H
}

// At devuelve el terreno de la celda p (p debe estar dentro del mapa).
func (g *Grid) At(p Point) Cell { return g.Cells[p.Y*g.W+p.X] }

// Set cambia el terreno de la celda p.
func (g *Grid) Set(p Point, c Cell) { g.Cells[p.Y*g.W+p.X] = c }

// Walkable indica si se puede pisar la celda p (dentro del mapa y no es muro).
func (g *Grid) Walkable(p Point) bool {
	return g.InBounds(p) && g.At(p) != Wall
}

// Step es una transición válida: ir a la celda To con un costo Cost.
type Step struct {
	To   Point
	Cost float64
}

var (
	dirs4 = []Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	dirsD = []Point{{1, -1}, {1, 1}, {-1, 1}, {-1, -1}}
)

// StepCost es el costo de moverse de la celda a hacia la vecina b:
// costo del terreno de destino, multiplicado por raíz de 2 si el paso es diagonal.
func (g *Grid) StepCost(a, b Point) float64 {
	c := g.At(b).Cost()
	if a.X != b.X && a.Y != b.Y {
		c *= math.Sqrt2
	}
	return c
}

// Neighbors genera las transiciones válidas desde p.
// Con diag=false solo hay 4 direcciones; con diag=true hay 8, pero una diagonal
// solo se permite si las dos celdas laterales están libres (no se "corta" una esquina).
func (g *Grid) Neighbors(p Point, diag bool) []Step {
	out := make([]Step, 0, 8)
	for _, d := range dirs4 {
		q := Point{p.X + d.X, p.Y + d.Y}
		if g.Walkable(q) {
			out = append(out, Step{q, g.StepCost(p, q)})
		}
	}
	if diag {
		for _, d := range dirsD {
			q := Point{p.X + d.X, p.Y + d.Y}
			if g.Walkable(q) && g.Walkable(Point{p.X + d.X, p.Y}) && g.Walkable(Point{p.X, p.Y + d.Y}) {
				out = append(out, Step{q, g.StepCost(p, q)})
			}
		}
	}
	return out
}

// LineOfSight indica si desde la celda a se puede ver la celda b, es decir, si
// ninguna celda intermedia es muro. Usa el algoritmo de Bresenham para recorrer
// las celdas que cruza la línea recta entre ambos puntos.
func (g *Grid) LineOfSight(a, b Point) bool {
	dx := abs(b.X - a.X)
	dy := -abs(b.Y - a.Y)
	sx, sy := 1, 1
	if a.X > b.X {
		sx = -1
	}
	if a.Y > b.Y {
		sy = -1
	}
	err := dx + dy
	x, y := a.X, a.Y
	for {
		if (x != a.X || y != a.Y) && !g.Walkable(Point{x, y}) {
			return false
		}
		if x == b.X && y == b.Y {
			return true
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
