// Paquete search implementa las estrategias de búsqueda que usa el guardia:
//
//	BFS     búsqueda en anchura (ciega, cola FIFO)
//	UCS     costo uniforme / Dijkstra (ciega, cola de prioridad por g(n))
//	Greedy  primero el mejor (heurística, cola de prioridad por h(n))
//	A*      A estrella (heurística, cola de prioridad por f(n) = g(n) + h(n))
//
// Todas comparten el mismo esqueleto: una FRONTERA de nodos por explorar, un
// conjunto de nodos ya expandidos y una tabla cameFrom para reconstruir el
// camino al final (rastreo inverso). Lo único que cambia es cómo se ordena la
// frontera. Además, cada ejecución devuelve métricas para poder compararlas.
package search

import (
	"container/heap"
	"math"
	"time"

	"guardia/world"
)

// Algorithm identifica una estrategia de búsqueda.
type Algorithm int

const (
	BFS Algorithm = iota
	UCS
	Greedy
	AStar
)

// AllAlgorithms lista los algoritmos en el orden en que se muestran y comparan.
var AllAlgorithms = []Algorithm{BFS, UCS, Greedy, AStar}

// String devuelve el nombre completo del algoritmo.
func (a Algorithm) String() string {
	switch a {
	case BFS:
		return "BFS"
	case UCS:
		return "UCS (Dijkstra)"
	case Greedy:
		return "Greedy"
	case AStar:
		return "A*"
	}
	return "?"
}

// Short devuelve un nombre corto para tablas.
func (a Algorithm) Short() string {
	if a == UCS {
		return "UCS"
	}
	return a.String()
}

// Result contiene el camino encontrado y las métricas de la ejecución.
type Result struct {
	Algo        Algorithm
	Found       bool
	Path        []world.Point // de la celda inicial a la meta (ambas incluidas)
	Cost        float64       // costo real g(meta) del camino devuelto
	Expanded    int           // nodos expandidos (sacados de la frontera y procesados)
	Generated   int           // nodos insertados en la frontera
	MaxFrontier int           // tamaño máximo de la frontera (memoria)
	Duration    time.Duration // tiempo de CPU de la búsqueda
	Visited     []world.Point // nodos expandidos, EN ORDEN (para dibujarlos)
}

// Steps devuelve la cantidad de pasos del camino (nodos - 1).
func (r Result) Steps() int {
	if len(r.Path) == 0 {
		return 0
	}
	return len(r.Path) - 1
}

// Heuristic estima el costo restante de a hacia b, h(n).
//
// Con 4 direcciones usa la distancia Manhattan. Con 8 direcciones usa la
// distancia octil (dx + dy - (2 - raíz2) * min(dx, dy)). Como el costo mínimo
// de cualquier paso es 1, ambas NUNCA sobreestiman: son admisibles (y
// consistentes), por lo que A* devuelve siempre un camino óptimo.
func Heuristic(a, b world.Point, diag bool) float64 {
	dx := math.Abs(float64(a.X - b.X))
	dy := math.Abs(float64(a.Y - b.Y))
	if !diag {
		return dx + dy
	}
	return (dx + dy) + (math.Sqrt2-2)*math.Min(dx, dy)
}

// Run busca un camino de start a goal con el algoritmo indicado.
func Run(g *world.Grid, start, goal world.Point, diag bool, algo Algorithm) Result {
	t0 := time.Now()
	var res Result
	if g.Walkable(start) && g.Walkable(goal) {
		if algo == BFS {
			res = bfs(g, start, goal, diag)
		} else {
			res = bestFirst(g, start, goal, diag, algo)
		}
	}
	res.Algo = algo
	res.Duration = time.Since(t0)
	return res
}

// RunAvg repite la búsqueda reps veces y devuelve el último resultado con el
// tiempo PROMEDIO. Una sola ejecución dura microsegundos y es muy ruidosa;
// promediar da cifras comparables.
func RunAvg(g *world.Grid, start, goal world.Point, diag bool, algo Algorithm, reps int) Result {
	if reps < 1 {
		reps = 1
	}
	var total time.Duration
	var res Result
	for i := 0; i < reps; i++ {
		res = Run(g, start, goal, diag, algo)
		total += res.Duration
	}
	res.Duration = total / time.Duration(reps)
	return res
}

// PathCost suma el costo real de recorrer un camino sobre el mapa.
func PathCost(g *world.Grid, path []world.Point) float64 {
	total := 0.0
	for i := 1; i < len(path); i++ {
		total += g.StepCost(path[i-1], path[i])
	}
	return total
}

// rebuild hace el RASTREO INVERSO: parte de la meta y sigue la tabla cameFrom
// (hijo -> padre) hasta el inicio; luego invierte el resultado.
func rebuild(cameFrom map[world.Point]world.Point, start, goal world.Point) []world.Point {
	path := []world.Point{goal}
	for cur := goal; cur != start; {
		cur = cameFrom[cur]
		path = append(path, cur)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// bfs: frontera = cola FIFO. Explora por niveles (todos los nodos a 1 paso,
// luego a 2 pasos, ...). Minimiza el NÚMERO de pasos e ignora los costos.
func bfs(g *world.Grid, start, goal world.Point, diag bool) Result {
	var r Result
	cameFrom := map[world.Point]world.Point{start: start} // también hace de "visto"
	queue := []world.Point{start}
	r.Generated, r.MaxFrontier = 1, 1

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		r.Expanded++
		r.Visited = append(r.Visited, cur)

		if cur == goal {
			r.Found = true
			r.Path = rebuild(cameFrom, start, goal)
			r.Cost = PathCost(g, r.Path)
			return r
		}
		for _, st := range g.Neighbors(cur, diag) {
			if _, seen := cameFrom[st.To]; seen {
				continue
			}
			cameFrom[st.To] = cur
			queue = append(queue, st.To)
			r.Generated++
		}
		if len(queue) > r.MaxFrontier {
			r.MaxFrontier = len(queue)
		}
	}
	return r
}

// item es un elemento de la cola de prioridad.
type item struct {
	p    world.Point
	prio float64 // criterio principal (depende del algoritmo)
	tie  float64 // desempate: entre igual prioridad, gana el de menor valor
	seq  int     // último desempate: orden de inserción (resultados deterministas)
}

// pq implementa heap.Interface (montículo mínimo).
type pq []item

func (q pq) Len() int { return len(q) }
func (q pq) Less(i, j int) bool {
	if math.Abs(q[i].prio-q[j].prio) > 1e-9 {
		return q[i].prio < q[j].prio
	}
	if math.Abs(q[i].tie-q[j].tie) > 1e-9 {
		return q[i].tie < q[j].tie
	}
	return q[i].seq < q[j].seq
}
func (q pq) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *pq) Push(x any)   { *q = append(*q, x.(item)) }
func (q *pq) Pop() any {
	old := *q
	n := len(old)
	it := old[n-1]
	*q = old[:n-1]
	return it
}

// bestFirst implementa UCS, Greedy y A*. La única diferencia es la prioridad:
//
//	UCS     prioridad = g(n)
//	Greedy  prioridad = h(n)
//	A*      prioridad = f(n) = g(n) + h(n)   (desempate: menor h)
//
// g(n) es el costo real acumulado desde el inicio y h(n) la estimación hasta la meta.
func bestFirst(g *world.Grid, start, goal world.Point, diag bool, algo Algorithm) Result {
	var r Result
	priority := func(p world.Point, gc float64) (prio, tie float64) {
		h := Heuristic(p, goal, diag)
		switch algo {
		case UCS:
			return gc, 0
		case Greedy:
			return h, 0
		default: // A*
			return gc + h, h
		}
	}

	gScore := map[world.Point]float64{start: 0}           // mejor g(n) conocido
	cameFrom := map[world.Point]world.Point{start: start} // padre de cada nodo
	closed := map[world.Point]bool{}                      // nodos ya expandidos
	open := &pq{}
	seq := 0

	pr, tie := priority(start, 0)
	heap.Push(open, item{start, pr, tie, seq})
	seq++
	r.Generated, r.MaxFrontier = 1, 1

	for open.Len() > 0 {
		cur := heap.Pop(open).(item)
		if closed[cur.p] {
			continue // entrada obsoleta: el nodo ya se expandió por un camino mejor
		}
		closed[cur.p] = true
		r.Expanded++
		r.Visited = append(r.Visited, cur.p)

		if cur.p == goal {
			r.Found = true
			r.Path = rebuild(cameFrom, start, goal)
			r.Cost = PathCost(g, r.Path)
			return r
		}
		for _, st := range g.Neighbors(cur.p, diag) {
			if closed[st.To] {
				continue
			}
			ng := gScore[cur.p] + st.Cost
			old, seen := gScore[st.To]
			if algo == Greedy {
				if seen {
					continue // Greedy no corrige: se queda con el primer padre
				}
			} else if seen && ng >= old-1e-12 {
				continue // ya conocemos un camino igual o mejor a este nodo
			}
			gScore[st.To] = ng
			cameFrom[st.To] = cur.p
			pr, tie := priority(st.To, ng)
			heap.Push(open, item{st.To, pr, tie, seq})
			seq++
			r.Generated++
		}
		// Frontera real = nodos descubiertos que aún no se expanden.
		if f := len(gScore) - len(closed); f > r.MaxFrontier {
			r.MaxFrontier = f
		}
	}
	return r
}
