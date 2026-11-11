// Comando bench: compara BFS, UCS, Greedy y A* en todos los mapas, SIN interfaz
// gráfica. Imprime una tabla y guarda resultados_benchmark.csv como evidencia
// para la sección "Comparativa de eficiencia entre estrategias" del reporte.
//
// Uso:
//
//	go run ./cmd/bench            # 4 direcciones
//	go run ./cmd/bench -diag      # 8 direcciones
//	go run ./cmd/bench -reps 1000 # más repeticiones para promediar el tiempo
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"guardia/search"
	"guardia/world"
)

func main() {
	diag := flag.Bool("diag", false, "permitir movimiento en diagonal (8 direcciones)")
	reps := flag.Int("reps", 500, "repeticiones por búsqueda para promediar el tiempo")
	out := flag.String("csv", "resultados_benchmark.csv", "archivo CSV de salida")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write([]string{"mapa", "algoritmo", "diagonales", "encontrado", "pasos", "costo",
		"expandidos", "generados", "frontera_max", "tiempo_us"})

	fmt.Printf("Diagonales: %v | repeticiones por búsqueda: %d\n", *diag, *reps)
	for _, lv := range world.Levels() {
		fmt.Printf("\n=== %s: %s\n", lv.Name, lv.Description)
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "Algoritmo\tPasos\tCosto\tExpandidos\tGenerados\tFrontera máx\tTiempo (µs)\t")

		var optimal float64
		results := make([]search.Result, 0, 4)
		for _, a := range search.AllAlgorithms {
			r := search.RunAvg(lv.Grid, lv.GuardStart, lv.PlayerStart, *diag, a, *reps)
			results = append(results, r)
			if a == search.UCS {
				optimal = r.Cost
			}
		}
		for _, r := range results {
			mark := ""
			if r.Found && r.Cost > optimal+1e-6 {
				mark = " (no óptimo)"
			}
			us := float64(r.Duration.Nanoseconds()) / 1000
			fmt.Fprintf(tw, "%s\t%d\t%.2f%s\t%d\t%d\t%d\t%.1f\t\n",
				r.Algo.Short(), r.Steps(), r.Cost, mark, r.Expanded, r.Generated, r.MaxFrontier, us)
			w.Write([]string{lv.Name, r.Algo.Short(), fmt.Sprint(*diag), fmt.Sprint(r.Found),
				fmt.Sprint(r.Steps()), fmt.Sprintf("%.2f", r.Cost), fmt.Sprint(r.Expanded),
				fmt.Sprint(r.Generated), fmt.Sprint(r.MaxFrontier), fmt.Sprintf("%.1f", us)})
		}
		tw.Flush()
	}
	fmt.Printf("\nResultados guardados en %s\n", *out)
}
