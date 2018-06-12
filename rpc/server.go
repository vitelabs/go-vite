package rpc

import (
	"net/http"
	"log"
)

var (
	host = "127.0.0.1"
	port  = "8080"
)


func StartServer () {
	router := GetRouter()

	log.Fatal(http.ListenAndServe(host + ":" + port, router))
}