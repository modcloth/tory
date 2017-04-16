package main

import (
	"log"
	"net/http"
	"os"

	"github.com/meatballhat/maybestatic"
)

func main() {
	http.Handle("/", http.FileServer(maybestatic.New("public", Asset)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "9375"
	}

	log.Printf("Listening at :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
