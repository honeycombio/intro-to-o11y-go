package main

import (
  "net/http"
  "fmt"
  "log"
)

func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Ohai")
  })

  err := http.ListenAndServe(":3000", nil)
  if err != nil {
    log.Fatalf("Could not start web server: %s", err)
  }
}