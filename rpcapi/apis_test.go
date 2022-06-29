package rpcapi

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/v2"
	"github.com/vitelabs/go-vite/v2/rpc"
	"github.com/vitelabs/go-vite/v2/rpcapi/api/filters"
)

type Service struct{}

func TestApiTypeValues(t *testing.T) {
	expected := apiTypeLimit
	actual := len(apiTypeStrings)
	if actual != expected {
		t.Fatalf("length of apiTypeStrings %v does not match with apiTypeLimit %v", actual, expected)
	}
}

func TestApiTypeName(t *testing.T) {
	expected1 := "health"
	actual1 := ApiType(HEALTH).name()
	if actual1 != expected1 {
		t.Fatalf("expected ApiType name: %s / actual: %s", expected1, actual1)
	}

	expected2 := "virtual"
	actual2 := ApiType(VIRTUAL).name()
	if actual2 != expected2 {
		t.Fatalf("expected ApiType name: %s / actual: %s", expected2, actual2)
	}
}

func TestPrintAllApiTypes(t *testing.T) {
	for at := ApiType(0); at < apiTypeLimit; at++ {
		fmt.Println(at.name())
	}
}

func createServer(t *testing.T) *rpc.Server {
	viteServer, err := vite.NewMock(nil, nil)
	if err != nil {
		t.Fatalf("create viteServer failed, %v", err)
	}
	filters.Es = filters.NewEventSystem(viteServer)

	server := rpc.NewServer()

	for at := ApiType(0); at < apiTypeLimit; at++ {
		api := GetApi(viteServer, at.name())
		server.RegisterName(api.Namespace, api.Service)
	}

	return server
}

func TestPrintAllExposedApiMethods(t *testing.T) {
	server := createServer(t)

	for svcName, methods := range server.Methods() {
		for m := range methods {
			fmt.Printf("%s_%s\n", svcName, methods[m])
		}
	}
}

func TestPrintAllExposedApiSubscriptions(t *testing.T) {
	server := createServer(t)

	for svcName, subs := range server.Subscriptions() {
		for s := range subs {
			fmt.Printf("%s_%s\n", svcName, subs[s])
		}
	}
}
