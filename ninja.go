package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var addr = flag.String("addr", ":3001", "address")

func handler(w http.ResponseWriter, r *http.Request) {
	log.SetPrefix("HTTP")
	log.Printf("%s %s", r.Method, r.RequestURI)
	fmt.Fprintf(w, "His foo, I love %s!", r.URL.Path[1:])
}

func main() {
	var port = os.Getenv("PORT")
	fmt.Println("PORT Z1Xs", ":"+port)

	http.HandleFunc("/", handler)
	log.Println("sup?")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
