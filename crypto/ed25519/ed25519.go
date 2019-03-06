// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ed25519 implements the Ed25519 signature algorithm. See
// https://ed25519.cr.yp.to/.
//
// These functions are also compatible with the “Ed25519” function defined in
// RFC 8032.
package ed25519

// This code is a port of the public domain, “ref10” implementation of ed25519
// from SUPERCOP.

// Be Noted that This Ed25519 uses Blake2b instead of Sha512

import (
	"bytes"
	"crypto"

	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/vitelabs/go-vite/crypto/ed25519/internal/edwards25519"
	"golang.org/x/crypto/blake2b"
)

const (
	// PublicKeySize is the size, in bytes, of public keys as used in this package.
	PublicKeySize = 32
	// PrivateKeySize is the size, in bytes, of private keys as used in this package.
	PrivateKeySize = 64
	// SignatureSize is the size, in bytes, of signatures generated and verified by this package.
	SignatureSize = 64

	X25519SkSize = 32

	dummyMessage = "vite is best"
)

// PublicKey is the type of Ed25519 public keys.
type PublicKey []byte

// PrivateKey is the type of Ed25519 private keys. It implements crypto.Signer.
type PrivateKey []byte

// Public returns the PublicKey corresponding to priv.
func (priv PrivateKey) Public() crypto.PublicKey {
	return PublicKey(priv.PubByte())
}

// Public returns the PublicKey in []byte type corresponding to priv.
func (priv PrivateKey) PubByte() []byte {
	publicKey := make([]byte, PublicKeySize)
	copy(publicKey, priv[32:])
	return publicKey
}

// Public returns the PublicKey in []byte type corresponding to priv.
func (priv PrivateKey) ToX25519Sk() []byte {
	digest := blake2b.Sum512(priv[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	return digest[:32]
}

func (priv PrivateKey) Hex() string {
	return hex.EncodeToString(priv)
}

func (pub PublicKey) Hex() string {
	return hex.EncodeToString(pub)
}

func (pub PublicKey) ToX25519Pk() []byte {
	/**
		ge25519_p3 A;
		fe25519    x;
		fe25519    one_minus_y;

		if (ge25519_has_small_order(ed25519_pk) != 0 ||
			ge25519_frombytes_negate_vartime(&A, ed25519_pk) != 0 ||
			ge25519_is_on_main_subgroup(&A) == 0) {
			return -1;
		}
		fe25519_1(one_minus_y);
		fe25519_sub(one_minus_y, one_minus_y, A.Y);
		fe25519_1(x);
		fe25519_add(x, x, A.Y);
		fe25519_invert(one_minus_y, one_minus_y);
		fe25519_mul(x, x, one_minus_y);
		fe25519_tobytes(curve25519_pk, x);

		return 0;
	 */

	var A edwards25519.ExtendedGroupElement
	var x edwards25519.FieldElement
	var one_minus_y edwards25519.FieldElement

	var p32 [32]byte
	copy(p32[:], pub)
	A.FromBytes(&p32)

	edwards25519.FeOne(&one_minus_y)
	edwards25519.FeSub(&one_minus_y, &one_minus_y, &A.Y)
	edwards25519.FeOne(&x)
	edwards25519.FeAdd(&x, &x, &A.Y)
	edwards25519.FeInvert(&one_minus_y, &one_minus_y)
	edwards25519.FeMul(&x, &x, &one_minus_y)

	var s [32]byte
	edwards25519.FeToBytes(&s, &x)
	return s[:]
}




func HexToPublicKey(hexstr string) (PublicKey, error) {
	b, e := hex.DecodeString(hexstr)
	if e != nil {
		return nil, e
	}
	if len(b) != PublicKeySize {
		return nil, fmt.Errorf("wrong pubkey size %v", len(b))
	}
	return b, nil
}

func HexToPrivateKey(hexstr string) (PrivateKey, error) {
	b, e := hex.DecodeString(hexstr)
	if e != nil {
		return nil, e
	}
	if len(b) != PrivateKeySize {
		return nil, fmt.Errorf("wrong private key size %v", len(b))
	}
	return b, nil
}

func IsValidPrivateKey(priv PrivateKey) bool {
	if l := len(priv); l != PrivateKeySize {
		return false
	}

	if Verify(priv.PubByte(), []byte(dummyMessage), Sign(priv, []byte(dummyMessage))) {
		return true
	}
	return false
}

func (pri PrivateKey) Clear() {
	for i := range pri {
		pri[i] = 0
	}
}

// Sign signs the given message with priv.
// Ed25519 performs two passes over messages to be signed and therefore cannot
// handle pre-hashed messages. Thus opts.HashFunc() must return zero to
// indicate the message hasn't been hashed. This can be achieved by passing
// crypto.Hash(0) as the value for opts.
func (priv PrivateKey) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	if opts.HashFunc() != crypto.Hash(0) {
		return nil, errors.New("ed25519: cannot sign hashed message")
	}

	return Sign(priv, message), nil
}

// GenerateKey generates a public/private key pair using entropy from rand.
// If rand is nil, crypto/rand.Reader will be used.
func GenerateKey(rand io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
	if rand == nil {
		rand = cryptorand.Reader
	}

	var randD [32]byte

	_, err = io.ReadFull(rand, randD[:])
	if err != nil {
		return nil, nil, err
	}

	return GenerateKeyFromD(randD)
}

// GenerateKeyFromD generates a public/private key pair using user given Deterministic value called d
func GenerateKeyFromD(d [32]byte) (publicKey PublicKey, privateKey PrivateKey, err error) {
	privateKey = make([]byte, PrivateKeySize)
	copy(privateKey[:32], d[:])

	publicKey = make([]byte, PublicKeySize)

	digest := blake2b.Sum512(privateKey[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	var A edwards25519.ExtendedGroupElement
	var hBytes [32]byte
	copy(hBytes[:], digest[:])
	edwards25519.GeScalarMultBase(&A, &hBytes)
	var publicKeyBytes [32]byte
	A.ToBytes(&publicKeyBytes)

	copy(privateKey[32:], publicKeyBytes[:])
	copy(publicKey, publicKeyBytes[:])

	return publicKey, privateKey, nil
}

// Sign signs the message with privateKey and returns a signature. It will
// panic if len(privateKey) is not PrivateKeySize.
func Sign(privateKey PrivateKey, message []byte) []byte {
	if l := len(privateKey); l != PrivateKeySize {
		panic("ed25519: bad private key length: " + strconv.Itoa(l))
	}

	h, err := blake2b.New512(nil)
	if err != nil {
		panic("ed25519: blake2b New512 Fail in Sign: " + err.Error())
	}

	h.Write(privateKey[:32])

	var digest1, messageDigest, hramDigest [64]byte
	var expandedSecretKey [32]byte
	h.Sum(digest1[:0])
	copy(expandedSecretKey[:], digest1[:])
	expandedSecretKey[0] &= 248
	expandedSecretKey[31] &= 63
	expandedSecretKey[31] |= 64

	h.Reset()
	h.Write(digest1[32:])
	h.Write(message)
	h.Sum(messageDigest[:0])

	var messageDigestReduced [32]byte
	edwards25519.ScReduce(&messageDigestReduced, &messageDigest)
	var R edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&R, &messageDigestReduced)

	var encodedR [32]byte
	R.ToBytes(&encodedR)

	h.Reset()
	h.Write(encodedR[:])
	h.Write(privateKey[32:])
	h.Write(message)
	h.Sum(hramDigest[:0])
	var hramDigestReduced [32]byte
	edwards25519.ScReduce(&hramDigestReduced, &hramDigest)

	var s [32]byte
	edwards25519.ScMulAdd(&s, &hramDigestReduced, &expandedSecretKey, &messageDigestReduced)

	signature := make([]byte, SignatureSize)
	copy(signature[:], encodedR[:])
	copy(signature[32:], s[:])

	return signature
}

// Verify reports whether sig is a valid signature of message by publicKey. It
// will panic if len(publicKey) is not PublicKeySize.
func Verify(publicKey PublicKey, message, sig []byte) bool {
	if l := len(publicKey); l != PublicKeySize {
		panic("ed25519: bad public key length: " + strconv.Itoa(l))
	}

	if len(sig) != SignatureSize || sig[63]&224 != 0 {
		return false
	}

	var A edwards25519.ExtendedGroupElement
	var publicKeyBytes [32]byte
	copy(publicKeyBytes[:], publicKey)
	if !A.FromBytes(&publicKeyBytes) {
		return false
	}
	edwards25519.FeNeg(&A.X, &A.X)
	edwards25519.FeNeg(&A.T, &A.T)

	h, err := blake2b.New512(nil)
	if err != nil {
		panic("ed25519: blake2b New512 Fail in Verify : " + err.Error())
	}

	h.Write(sig[:32])
	h.Write(publicKey[:])
	h.Write(message)
	var digest [64]byte
	h.Sum(digest[:0])

	var hReduced [32]byte
	edwards25519.ScReduce(&hReduced, &digest)

	var R edwards25519.ProjectiveGroupElement
	var s [32]byte
	copy(s[:], sig[32:])

	// https://tools.ietf.org/html/rfc8032#section-5.1.7 requires that s be in
	// the range [0, order) in order to prevent signature malleability.
	if !edwards25519.ScMinimal(&s) {
		return false
	}

	edwards25519.GeDoubleScalarMultVartime(&R, &hReduced, &A, &s)

	var checkR [32]byte
	R.ToBytes(&checkR)
	return bytes.Equal(sig[:32], checkR[:])
}
