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
 * You should have received chain copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/vitelabs/go-vite/p2p/discovery"
)

func main() {
	pwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	cfg1, err := discovery.NewConfig("127.0.0.1:8483", "127.0.0.1:8483", filepath.Join(pwd, ".mock1"), "", nil, nil, 10)
	if err != nil {
		panic(err)
	}

	cfg2, err := discovery.NewConfig("127.0.0.1:8484", "127.0.0.1:8484", filepath.Join(pwd, ".mock2"), "", nil, nil, 10)
	if err != nil {
		panic(err)
	}

	cfg3, err := discovery.NewConfig("127.0.0.1:8485", "127.0.0.1:8485", filepath.Join(pwd, ".mock3"), "", nil, nil, 10)
	if err != nil {
		panic(err)
	}

	cfg4, err := discovery.NewConfig("127.0.0.1:8486", "127.0.0.1:8486", filepath.Join(pwd, ".mock4"), "", nil, nil, 10)
	if err != nil {
		panic(err)
	}

	cfg5, err := discovery.NewConfig("127.0.0.1:8487", "127.0.0.1:8487", filepath.Join(pwd, ".mock5"), "", nil, nil, 10)
	if err != nil {
		panic(err)
	}

	cfg1.BootNodes = append(cfg1.BootNodes, cfg2.Node().String())
	cfg2.BootNodes = append(cfg2.BootNodes, cfg3.Node().String())
	cfg3.BootNodes = append(cfg3.BootNodes, cfg4.Node().String())
	cfg4.BootNodes = append(cfg4.BootNodes, cfg5.Node().String())

	d1 := discovery.New(cfg1)
	d2 := discovery.New(cfg2)
	d3 := discovery.New(cfg3)
	d4 := discovery.New(cfg4)
	d5 := discovery.New(cfg5)

	start := func(d discovery.Discovery) {
		err = d.Start()
		if err != nil {
			panic(err)
		}
	}

	go start(d1)
	time.Sleep(time.Second)
	go start(d2)
	time.Sleep(time.Second)
	go start(d3)
	time.Sleep(time.Second)
	go start(d4)
	time.Sleep(time.Second)
	go start(d5)

	fmt.Println("start")

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				fmt.Println("d1", d1.AllNodes())
				fmt.Println("d2", d2.AllNodes())
				fmt.Println("d3", d3.AllNodes())
				fmt.Println("d4", d4.AllNodes())
				fmt.Println("d5", d5.AllNodes())
				fmt.Println("------------")
			}
		}
	}()

	err = http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		panic(err)
	}
}
