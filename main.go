package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	_ "embed"
)

// global variables are all but beautiful, but uh, I don't care for this lil evening project

//go:embed index.html
var htmlTemplate []byte
var tmpl, _ = template.New("bonk").Parse(string(htmlTemplate))

var count atomic.Int64

var addr string

func main() {
	fs := flag.NewFlagSet("bonkme", flag.ExitOnError)
	fs.StringVar(&addr, "addr", ":4000", "the address to listen on")
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("parsing arguments failed: %v", err)
	}

	if err := run(context.Background()); err != nil {
		log.Fatalf("run failed: %v", err)
	}
}

func run(ctx context.Context) error {
	f, err := os.OpenFile("bonks.csv", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("opening bonks log: %w", err)
	}
	defer f.Close()

	// set the global bonk counter
	count.Add(int64(readNewLines(f)))

	// setup our writer for new bonks
	bonks := csv.NewWriter(f)
	defer bonks.Flush()

	// We setup a mux that reacts to:
	// - GET / -- renders the template
	// - POST /bonk -- adds a bonk
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if err := tmpl.Execute(w, count.Load()); err != nil {
				w.Write([]byte("rendering the temolate failed, theoretically this itself qualifies as a bonk."))
				return
			}
		}

		if r.Method != http.MethodPost {
			return
		}

		err := bonks.Write([]string{
			time.Now().Format(time.RFC3339),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("appending bonk failed: %s", err)))
		}
		bonks.Flush()

		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<marquee>thanks for the bonk uwu</marquee>"))
		count.Add(1)
	})
	log.Printf("listening on: %s", addr)
	return http.ListenAndServe(addr, m)
}

func readNewLines(r io.Reader) int {
	buf := make([]byte, 2<<10)
	newLines := []byte("\n")
	var count int

	for {
		// errors do not matter, in the worst case we just return 0.
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], newLines)

		if errors.Is(err, io.EOF) {
			return count
		} else if err != nil {
			return 0
		}
	}
}
