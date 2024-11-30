package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/wrap", wrapHandler)
	http.HandleFunc("/unwrap", unwrapHandler)

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("Server starting on :3001")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatal(err)
	}
}