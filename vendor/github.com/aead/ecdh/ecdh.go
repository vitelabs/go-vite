// Copyright (c) 2016 Andreas Auernhammer. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

// Package ecdh implements the Diffie-Hellman key exchange
// using elliptic curves (ECDH). It directly provides ECDH
// implementations for the NIST curves P224, P256, P384,
// and Bernstein's Cruve25519.
//
// For generic curves this implementation of ECDH
// only uses the x-coordinate as the computed secret.
package ecdh // import "github.com/aead/ecdh"

import (
	"crypto"
	"io"
)

// KeyExchange is the interface defining all functions
// necessary for ECDH.
type KeyExchange interface {
	// GenerateKey generates a private/public key pair using entropy from rand.
	// If rand is nil, crypto/rand.Reader will be used.
	GenerateKey(rand io.Reader) (private crypto.PrivateKey, public crypto.PublicKey, err error)

	// Params returns the curve parameters - like the field size.
	Params() CurveParams

	// PublicKey returns the public key corresponding to the given private one.
	PublicKey(private crypto.PrivateKey) (public crypto.PublicKey)

	// Check returns a non-nil error if the peers public key cannot used for the
	// key exchange - for instance the public key isn't a point on the elliptic curve.
	// It's recommended to check peer's public key before computing the secret.
	Check(peersPublic crypto.PublicKey) (err error)

	// ComputeSecret returns the secret value computed from the given private key
	// and the peers public key.
	ComputeSecret(private crypto.PrivateKey, peersPublic crypto.PublicKey) (secret []byte)
}

// CurveParams contains the parameters of an elliptic curve.
type CurveParams struct {
	Name    string // the canonical name of the curve
	BitSize int    // the size of the underlying field
}
