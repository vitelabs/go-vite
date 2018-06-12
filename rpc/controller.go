package rpc

import (
	"net/http"
	"fmt"
	"github.com/julienschmidt/httprouter"
)

func HelloWorld(w http.ResponseWriter, r *http.Request, _ httprouter.Params)  {
	fmt.Fprint(w, "Welcome!\n")
}

