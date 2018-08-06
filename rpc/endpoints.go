package rpc

import (
	"net"
	"net/rpc"
	"github.com/inconshreveable/log15"
)

var rLog = log15.New("module", "rpc")

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Version   string      // api version for DApp's
	Service   interface{} // receiver instance which holds the methods
	Public    bool        // indication if the methods must be considered safe for public use
}

func StartIPCEndpoint(lis net.Listener, apis []API) (*rpc.Server, error) {
	srv := rpc.NewServer()
	for _, api := range apis {
		if err := srv.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, err
		}
		rLog.Debug("IPC registered", " namespace", api.Namespace, " Service", api.Service)
	}
	if err := ServeListener(srv, lis); err != nil {
		return nil, err
	}
	return srv, nil
}

func StartInProc(apis []API) (*rpc.Server, error) {
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, err
		}
	}
	return handler, nil
}
