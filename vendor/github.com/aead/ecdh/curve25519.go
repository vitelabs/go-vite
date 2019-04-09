// Copyright (c) 2016 Andreas Auernhammer. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package ecdh

import (
	"crypto"
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/curve25519"
)

type ecdh25519 struct{}

var curve25519Params = CurveParams{
	Name:    "Curve25519",
	BitSize: 255,
}

// X25519 creates a new ecdh.KeyExchange with
// the elliptic curve Curve25519.
func X25519() KeyExchange {
	return ecdh25519{}
}

func (ecdh25519) GenerateKey(random io.Reader) (private crypto.PrivateKey, public crypto.PublicKey, err error) {
	if random == nil {
		random = rand.Reader
	}

	var pri, pub [32]byte
	_, err = io.ReadFull(random, pri[:])
	if err != nil {
		return
	}

	// From https://cr.yp.to/ecdh.html
	pri[0] &= 248
	pri[31] &= 127
	pri[31] |= 64

	curve25519.ScalarBaseMult(&pub, &pri)

	private = pri
	public = pub
	return
}

func (ecdh25519) Params() CurveParams { return curve25519Params }

func (ecdh25519) PublicKey(private crypto.PrivateKey) (public crypto.PublicKey) {
	var pri, pub [32]byte
	if ok := checkType(&pri, private); !ok {
		panic("ecdh: unexpected type of private key")
	}

	curve25519.ScalarBaseMult(&pub, &pri)

	public = pub
	return
}

func (ecdh25519) Check(peersPublic crypto.PublicKey) (err error) {
	if ok := checkType(new([32]byte), peersPublic); !ok {
		err = errors.New("unexptected type of peers public key")
	}
	return
}

func (ecdh25519) ComputeSecret(private crypto.PrivateKey, peersPublic crypto.PublicKey) (secret []byte) {
	var sec, pri, pub [32]byte
	if ok := checkType(&pri, private); !ok {
		panic("ecdh: unexpected type of private key")
	}
	if ok := checkType(&pub, peersPublic); !ok {
		panic("ecdh: unexpected type of peers public key")
	}

	curve25519.ScalarMult(&sec, &pri, &pub)

	secret = sec[:]
	return
}

func checkType(key *[32]byte, typeToCheck interface{}) (ok bool) {
	switch t := typeToCheck.(type) {
	case [32]byte:
		copy(key[:], t[:])
		ok = true
	case *[32]byte:
		copy(key[:], t[:])
		ok = true
	case []byte:
		if len(t) == 32 {
			copy(key[:], t)
			ok = true
		}
	case *[]byte:
		if len(*t) == 32 {
			copy(key[:], *t)
			ok = true
		}
	}
	return
}
