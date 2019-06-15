/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package discovery

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

type mockHandler struct {
	request Request
	nodes   []string
	t       *testing.T
}

func (m *mockHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		r := req.Body
		defer r.Close()

		var buf []byte
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			m.t.Errorf("failed to read body: %v", err)
		}

		var request = new(Request)
		err = json.Unmarshal(buf, request)
		if err != nil {
			m.t.Errorf("failed to parse body: %v", err)
		}
		if false == request.Node.Equal(m.request.Node) {
			m.t.Errorf("node no equal: %v %v", m.request.Node, request.Node)
		}
		if request.Count != m.request.Count {
			m.t.Errorf("count not equal: %d %d", m.request.Count, request.Count)
		}

		buf, err = json.Marshal(Result{
			Code:    0,
			Message: "",
			Data:    m.nodes,
		})
		if err != nil {
			panic(err)
		}

		n, err := res.Write(buf)
		if err != nil {
			panic(err)
		}
		if n != len(buf) {
			panic("write too short")
		}
	} else {
		m.t.Errorf("unknown request method: %v", req.Method)
	}
}

func TestNetBooter(t *testing.T) {
	bootNodes := []string{
		"1111111111111111111111111111111111111111111111111111111111111111@111.111.111.111",
		"1111111111111111111111111111111111111111111111111111111111111110@111.111.111.112",
	}

	node := &vnode.Node{
		ID: vnode.ZERO,
		EndPoint: vnode.EndPoint{
			Host: []byte{127, 0, 0, 1},
			Port: 8888,
			Typ:  vnode.HostIPv4,
		},
		Net: 10,
		Ext: []byte{1, 2, 3},
	}

	const count = 2

	boot := newNetBooter(node, []string{"http://127.0.0.1:8080"})

	go func() {
		err := http.ListenAndServe("127.0.0.1:8080", &mockHandler{
			request: Request{
				Node:  node,
				Count: count,
			},
			nodes: bootNodes,
			t:     t,
		})
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Second)

	nodes := boot.getBootNodes(count)
	if len(nodes) != len(bootNodes) {
		t.Errorf("wrong bootnodes: %v", nodes)
	} else {
		for i, v := range nodes {
			if v.String() != bootNodes[i] {
				t.Errorf("different bootNode: %v %v", v.String(), bootNodes[i])
			}
		}
	}
}
