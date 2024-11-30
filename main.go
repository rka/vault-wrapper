package main

import (
	"net/http"
	"github.com/sirupsen/logrus"
)

func init() {
	// Set up logrus
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/wrap", wrapHandler)
	http.HandleFunc("/unwrap", unwrapHandler)

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	logrus.Info("Server starting on :3001")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		logrus.Fatal(err)
	}
}