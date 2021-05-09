// Program pokersplit is a web application to register cash game poker buy-ins
// and calculate who owns how much to whom at the end of the game.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/fhchstr/pokersplit/pokersplit/pokersplit"
)

var (
	port = flag.Int("port", 8080, "TCP port to listen on")
)

func main() {
	flag.Parse()
	http.HandleFunc("/", pokersplit.ServeHTTP)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
