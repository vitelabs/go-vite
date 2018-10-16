package node

import (
	"fmt"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi"
	"strings"
)

//In-proc apis
func (node *Node) GetInProcessApis() []rpc.API {
	return rpcapi.GetApis(node.viteServer, "ledger", "wallet", "private_onroad", "net", "contracts", "testapi", "pow", "tx")
}

//Ipc apis
func (node *Node) GetIpcApis() []rpc.API {
	return rpcapi.GetApis(node.viteServer, "ledger", "wallet", "private_onroad", "net", "contracts", "testapi", "pow", "tx")
}

//Http apis
func (node *Node) GetHttpApis() []rpc.API {
	apiModules := []string{"ledger", "public_onroad", "net", "contracts", "testapi", "pow", "tx"}
	if node.Config().NetID > 1 {
		apiModules = append(apiModules, "testapi")
	}
	return rpcapi.GetApis(node.viteServer, apiModules...)
}

//WS apis
func (node *Node) GetWSApis() []rpc.API {
	apiModules := []string{"ledger", "public_onroad", "net", "contracts", "testapi", "pow", "tx"}
	if node.Config().NetID > 1 {
		apiModules = append(apiModules, "testapi")
	}
	return rpcapi.GetApis(node.viteServer, apiModules...)
}

// startIPC initializes and starts the IPC RPC endpoint.
func (node *Node) startIPC(apis []rpc.API) error {
	if node.ipcEndpoint == "" {
		return nil // IPC disabled.
	}
	listener, handler, err := rpc.StartIPCEndpoint(node.ipcEndpoint, apis)
	if err != nil {
		return err
	}
	node.ipcListener = listener
	node.ipcHandler = handler
	log.Info("IPC endpoint opened", "url", node.ipcEndpoint)
	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (node *Node) stopIPC() {
	if node.ipcListener != nil {
		node.ipcListener.Close()
		node.ipcListener = nil

		log.Info("IPC endpoint closed", "endpoint", node.ipcEndpoint)
	}
	if node.ipcHandler != nil {
		node.ipcHandler.Stop()
		node.ipcHandler = nil
	}
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (node *Node) startHTTP(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string, timeouts rpc.HTTPTimeouts) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	listener, handler, err := rpc.StartHTTPEndpoint(endpoint, apis, modules, cors, vhosts, timeouts)
	if err != nil {
		return err
	}
	log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%s", endpoint), "cors", strings.Join(cors, ","), "vhosts", strings.Join(vhosts, ","))
	// All listeners booted successfully
	node.httpEndpoint = endpoint
	node.httpListener = listener
	node.httpHandler = handler

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (node *Node) stopHTTP() {
	if node.httpListener != nil {
		node.httpListener.Close()
		node.httpListener = nil

		log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%s", node.httpEndpoint))
	}
	if node.httpHandler != nil {
		node.httpHandler.Stop()
		node.httpHandler = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (node *Node) startWS(endpoint string, apis []rpc.API, modules []string, wsOrigins []string, exposeAll bool) error {
	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	listener, handler, err := rpc.StartWSEndpoint(endpoint, apis, modules, wsOrigins, exposeAll)
	if err != nil {
		return err
	}
	log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", listener.Addr()))
	// All listeners booted successfully
	node.wsEndpoint = endpoint
	node.wsListener = listener
	node.wsHandler = handler

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (node *Node) stopWS() {
	if node.wsListener != nil {
		node.wsListener.Close()
		node.wsListener = nil
		log.Info("WebSocket endpoint closed", "url", fmt.Sprintf("ws://%s", node.wsEndpoint))
	}
	if node.wsHandler != nil {
		node.wsHandler.Stop()
		node.wsHandler = nil
	}
}

func (node *Node) Attach() (*rpc.Client, error) {
	node.lock.RLock()
	defer node.lock.RUnlock()

	if node.p2pServer == nil {
		return nil, ErrNodeStopped
	}

	return rpc.DialInProc(node.inProcessHandler), nil
}

// startInProc initializes an in-process RPC endpoint.
func (node *Node) startInProcess(apis []rpc.API) error {
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		log.Debug("InProc registered", "service", api.Service, "namespace", api.Namespace)
	}
	node.inProcessHandler = handler
	return nil
}

// stopInProc terminates the in-process RPC endpoint.
func (node *Node) stopInProcess() {
	if node.inProcessHandler != nil {
		node.inProcessHandler.Stop()
		node.inProcessHandler = nil
	}
}
