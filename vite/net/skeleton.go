package net

import "github.com/vitelabs/go-vite/common/types"

type blockID struct {
	height uint64
	hash   types.Hash
	prev   types.Hash
}

// use to record blocks received

type skeleton struct {
	snapshotChain []*blockID
	accountChain  map[types.Address][]*blockID
	//index         map[types.Hash]uint64
}
