package vrpc

import "github.com/powerman/rpc-codec/jsonrpc2"

func JsonrpcClient(url string) *jsonrpc2.Client {
	return jsonrpc2.NewHTTPClient(url)
}
