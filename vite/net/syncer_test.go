package net

import (
	"testing"

	"github.com/vitelabs/go-vite/p2p"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite/net/message"
)

func TestSyncer_Handle(t *testing.T) {
	syncer := newSyncer(nil, newPeerSet(), nil, nil, nil)
	syncer.to = 100000

	// one
	fileList := &message.FileList{
		Files: []*ledger.CompressedFileMeta{
			{
				StartHeight: 0,
				EndHeight:   3599,
				Filename:    "file1",
			},
			{
				StartHeight: 3600,
				EndHeight:   7199,
				Filename:    "file2",
			},
		},
		Chunks: [][2]uint64{
			{7200, 100000},
		},
		Nonce: 0,
	}

	p := mockPeer()
	p.height = syncer.to + 1
	payload, err := fileList.Serialize()
	if err != nil {
		panic(err)
	}

	msg := p2p.Msg{
		CmdSet:  CmdSet,
		Cmd:     p2p.Cmd(FileListCode),
		Id:      0,
		Payload: payload,
	}
	syncer.Handle(&msg, p)

	// two
	fileList = &message.FileList{
		//Files: []*ledger.CompressedFileMeta{
		//	{
		//		StartHeight: 0,
		//		EndHeight:   3599,
		//		Filename:    "file1",
		//	},
		//	{
		//		StartHeight: 3600,
		//		EndHeight:   7199,
		//		Filename:    "file2",
		//	},
		//	{
		//		StartHeight: 7200,
		//		EndHeight:   10799,
		//		Filename:    "file3",
		//	},
		//},
		Chunks: [][2]uint64{
			{0, 100000},
		},
		Nonce: 0,
	}

	p = mockPeer()
	p.height = syncer.to + 1
	payload, err = fileList.Serialize()
	if err != nil {
		panic(err)
	}

	msg = p2p.Msg{
		CmdSet:  CmdSet,
		Cmd:     p2p.Cmd(FileListCode),
		Id:      0,
		Payload: payload,
	}
	syncer.Handle(&msg, p)

	if syncer.pool.queue.Size() != ((100001 / 50) + 1) {
		t.Fail()
	}

	// three
	fileList = &message.FileList{
		Chunks: [][2]uint64{
			{100, 100000},
		},
	}

	p = mockPeer()
	p.height = syncer.to + 1
	payload, err = fileList.Serialize()
	if err != nil {
		panic(err)
	}

	msg = p2p.Msg{
		CmdSet:  CmdSet,
		Cmd:     p2p.Cmd(FileListCode),
		Id:      0,
		Payload: payload,
	}
	syncer.Handle(&msg, p)

	if syncer.pool.queue.Size() != ((100001 / 50) + 1) {
		t.Fail()
	}
}
