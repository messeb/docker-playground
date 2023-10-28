package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		message := os.Getenv("INSTANCE")
		fmt.Fprint(w, "Hello, world! Instance: "+message+"\n")
	})

	http.ListenAndServe(":8080", nil)
}
