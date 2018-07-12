package rpc

import (
	"github.com/vitelabs/go-vite/log"
	"net"
	"net/rpc"
)

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Version   string      // api version for DApp's
	Service   interface{} // receiver instance which holds the methods
	Public    bool        // indication if the methods must be considered safe for public use
}

func StartIPCEndpoint(ipcEndpoint string, apis []API) (net.Listener, *rpc.Server, error) {
	srv := rpc.NewServer()
	for _, api := range apis {
		if err := srv.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, nil, err
		}
		log.Debug("IPC registered", "namespace", api.Namespace)
	}
	lis, err := ipcListen(ipcEndpoint)
	if err != nil {
		return nil, nil, err
	}
	if err = ServeListener(srv, lis); err != nil {
		return nil, nil, err
	}
	return lis, srv, nil
}
