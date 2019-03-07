// Copyright (c) 2016 Andreas Auernhammer. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package ecdh

import (
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"io"
	"math/big"
)

// Point represents a generic elliptic curve Point with a
// X and a Y coordinate.
type Point struct {
	X, Y *big.Int
}

// Generic creates a new ecdh.KeyExchange with
// generic elliptic.Curve implementations.
func Generic(c elliptic.Curve) KeyExchange {
	if c == nil {
		panic("ecdh: curve is nil")
	}
	return genericCurve{curve: c}
}

type genericCurve struct {
	curve elliptic.Curve
}

func (g genericCurve) GenerateKey(random io.Reader) (private crypto.PrivateKey, public crypto.PublicKey, err error) {
	if random == nil {
		random = rand.Reader
	}
	private, x, y, err := elliptic.GenerateKey(g.curve, random)
	if err != nil {
		private = nil
		return
	}
	public = Point{X: x, Y: y}
	return
}

func (g genericCurve) Params() CurveParams {
	p := g.curve.Params()
	return CurveParams{
		Name:    p.Name,
		BitSize: p.BitSize,
	}
}

func (g genericCurve) PublicKey(private crypto.PrivateKey) (public crypto.PublicKey) {
	key, ok := checkPrivateKey(private)
	if !ok {
		panic("ecdh: unexpected type of private key")
	}

	N := g.curve.Params().N
	if new(big.Int).SetBytes(key).Cmp(N) >= 0 {
		panic("ecdh: private key cannot used with given curve")
	}

	x, y := g.curve.ScalarBaseMult(key)
	public = Point{X: x, Y: y}
	return
}

func (g genericCurve) Check(peersPublic crypto.PublicKey) (err error) {
	key, ok := checkPublicKey(peersPublic)
	if !ok {
		err = errors.New("unexpected type of peers public key")
	}
	if !g.curve.IsOnCurve(key.X, key.Y) {
		err = errors.New("peer's public key is not on curve")
	}
	return
}

func (g genericCurve) ComputeSecret(private crypto.PrivateKey, peersPublic crypto.PublicKey) (secret []byte) {
	priKey, ok := checkPrivateKey(private)
	if !ok {
		panic("ecdh: unexpected type of private key")
	}
	pubKey, ok := checkPublicKey(peersPublic)
	if !ok {
		panic("ecdh: unexpected type of peers public key")
	}

	sX, _ := g.curve.ScalarMult(pubKey.X, pubKey.Y, priKey)

	secret = sX.Bytes()
	return
}

func checkPrivateKey(typeToCheck interface{}) (key []byte, ok bool) {
	switch t := typeToCheck.(type) {
	case []byte:
		key = t
		ok = true
	case *[]byte:
		key = *t
		ok = true
	}
	return
}

func checkPublicKey(typeToCheck interface{}) (key Point, ok bool) {
	switch t := typeToCheck.(type) {
	case Point:
		key = t
		ok = true
	case *Point:
		key = *t
		ok = true
	}
	return
}
