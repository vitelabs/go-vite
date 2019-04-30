// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package leveldb

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/comparer"
)

type IComparer struct {
	ucmp comparer.Comparer
}

func NewIComparer(ucmp comparer.Comparer) *IComparer {
	return &IComparer{
		ucmp: ucmp,
	}
}

func (icmp *IComparer) uName() string {
	return icmp.ucmp.Name()
}

func (icmp *IComparer) uCompare(a, b []byte) int {
	return icmp.ucmp.Compare(a, b)
}

func (icmp *IComparer) uSeparator(dst, a, b []byte) []byte {
	return icmp.ucmp.Separator(dst, a, b)
}

func (icmp *IComparer) uSuccessor(dst, b []byte) []byte {
	return icmp.ucmp.Successor(dst, b)
}

func (icmp *IComparer) Name() string {
	return icmp.uName()
}

func (icmp *IComparer) Compare(a, b []byte) int {
	x := icmp.uCompare(internalKey(a).ukey(), internalKey(b).ukey())
	if x == 0 {
		if m, n := internalKey(a).num(), internalKey(b).num(); m > n {
			return -1
		} else if m < n {
			return 1
		}
	}
	return x
}

func (icmp *IComparer) Separator(dst, a, b []byte) []byte {
	ua, ub := internalKey(a).ukey(), internalKey(b).ukey()
	dst = icmp.uSeparator(dst, ua, ub)
	if dst != nil && len(dst) < len(ua) && icmp.uCompare(ua, dst) < 0 {
		// Append earliest possible number.
		return append(dst, keyMaxNumBytes...)
	}
	return nil
}

func (icmp *IComparer) Successor(dst, b []byte) []byte {
	ub := internalKey(b).ukey()
	dst = icmp.uSuccessor(dst, ub)
	if dst != nil && len(dst) < len(ub) && icmp.uCompare(ub, dst) < 0 {
		// Append earliest possible number.
		return append(dst, keyMaxNumBytes...)
	}
	return nil
}
