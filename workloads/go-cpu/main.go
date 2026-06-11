package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

// fibonacci computes fib(n) recursively — intentionally expensive
// this makes each request CPU bound
func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// compute fib(30) — takes ~5ms of CPU per request
		result := fibonacci(30)
		fmt.Fprintf(w, "%d\n", result)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "healthy")
	})

	// allow configurable fibonacci depth via env for flexibility
	depth := 30
	if d := os.Getenv("FIB_DEPTH"); d != "" {
		if n, err := strconv.Atoi(d); err == nil {
			depth = n
		}
	}
	_ = depth

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
