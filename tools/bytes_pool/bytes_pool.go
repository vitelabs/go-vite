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

package bytes_pool

import "sync"

const bufSize0 = 128
const bufSize1 = 512
const bufSize2 = 10240  // 10k
const bufSize3 = 102400 // 100k

const maxTry = 3

type bytesPool struct {
	sync.Pool
	max int
}

var bytesPools = [...]bytesPool{
	{
		sync.Pool{
			New: func() interface{} {
				buf := make([]byte, bufSize0)
				return buf
			},
		},
		bufSize0,
	},
	{
		sync.Pool{
			New: func() interface{} {
				buf := make([]byte, bufSize1)
				return buf
			},
		},
		bufSize1,
	},
	{
		sync.Pool{
			New: func() interface{} {
				buf := make([]byte, bufSize2)
				return buf
			},
		},
		bufSize2,
	},
	{
		sync.Pool{
			New: func() interface{} {
				buf := make([]byte, bufSize3)
				return buf
			},
		},
		bufSize3,
	},
}

func Get(n int) []byte {
	for _, p := range bytesPools {
		if n < p.max {
			for i := 0; i < maxTry; i++ {
				buf := p.Get().([]byte)
				if cap(buf) >= n {
					return buf[:n]
				}
				p.Put(buf)
			}

			return make([]byte, n)
		}
	}

	return make([]byte, n)
}

func Put(buf []byte) {
	for _, p := range bytesPools {
		if cap(buf) < p.max {
			buf = buf[:cap(buf)]
			p.Put(buf)
		}
	}
}
