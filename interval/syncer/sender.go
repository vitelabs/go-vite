package syncer

import (
	"encoding/json"

	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/monitor"
	"github.com/vitelabs/go-vite/interval/p2p"
)

type sender struct {
	net p2p.P2P
}

func (ser *sender) broadcastState(s stateMsg) error {
	bytM, err := json.Marshal(&s)
	msg := p2p.NewMsg(common.State, bytM)

	if err != nil {
		return errors.New("broadcastState, format fail. err:" + err.Error())
	}
	peers, err := ser.net.AllPeer()
	if err != nil {
		log.Error("broadcastState, can't get all peer.%v", err)
		return err
	}
	if len(peers) == 0 {
		//log.Info("broadcast peer list is empty.")
		return nil
	}

	for _, p := range peers {
		tmpE := p.Write(msg)
		if tmpE != nil {
			err = tmpE
			log.Error("broadcastState, write data fail, peerId:%s, err:%v", p.Id(), err)
		}
	}
	return err
}

func (ser *sender) BroadcastAccountBlocks(address string, blocks []*common.AccountStateBlock) error {
	defer monitor.LogTime("net", "broadcastAccount", time.Now())
	bytM, err := json.Marshal(&accountBlocksMsg{Address: address, Blocks: blocks})
	msg := p2p.NewMsg(common.AccountBlocks, bytM)

	if err != nil {
		return errors.New("BroadcastAccountBlocks, format fail. err:" + err.Error())
	}
	peers, err := ser.net.AllPeer()
	if err != nil {
		log.Error("BroadcastAccountBlocks, can't get all peer.%v", err)
		return err
	}
	if len(peers) == 0 {
		//log.Info("broadcast peer list is empty.")
		return nil
	}

	for _, p := range peers {
		tmpE := p.Write(msg)
		if tmpE != nil {
			err = tmpE
			log.Error("BroadcastAccountBlocks, write data fail, peerId:%s, err:%v", p.Id(), err)
		}
	}
	return err
}

func (ser *sender) BroadcastSnapshotBlocks(blocks []*common.SnapshotBlock) error {
	defer monitor.LogTime("net", "broadcastSnapshot", time.Now())
	bytM, err := json.Marshal(&snapshotBlocksMsg{Blocks: blocks})
	msg := p2p.NewMsg(common.SnapshotBlocks, bytM)

	if err != nil {
		return errors.New("BroadcastSnapshotBlocks, format fail. err:" + err.Error())
	}
	peers, err := ser.net.AllPeer()
	if err != nil {
		log.Error("BroadcastSnapshotBlocks, can't get all peer.%v", err)
		return err
	}
	if len(peers) == 0 {
		log.Info("broadcast peer list is empty.")
		return nil
	}

	for _, p := range peers {
		tmpE := p.Write(msg)
		if tmpE != nil {
			err = tmpE
			log.Error("BroadcastSnapshotBlocks, write data fail, peerId:%s, err:%v", p.Id(), err)
		}
	}
	return err
}

func (ser *sender) SendAccountBlocks(address string, blocks []*common.AccountStateBlock, peer p2p.Peer) error {
	bytM, err := json.Marshal(&accountBlocksMsg{Address: address, Blocks: blocks})
	if err != nil {
		return errors.New("SendAccountBlocks, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.AccountBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("SendAccountBlocks, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (ser *sender) SendSnapshotBlocks(blocks []*common.SnapshotBlock, peer p2p.Peer) error {
	bytM, err := json.Marshal(&snapshotBlocksMsg{Blocks: blocks})
	if err != nil {
		return errors.New("SendSnapshotBlocks, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.SnapshotBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("SendSnapshotBlocks, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err

}

func (ser *sender) SendAccountHashes(address string, hashes []common.HashHeight, peer p2p.Peer) error {
	bytM, err := json.Marshal(&accountHashesMsg{Address: address, Hashes: hashes})
	if err != nil {
		return errors.New("SendAccountHashes, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.AccountHashes, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("SendAccountHashes, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (ser *sender) SendSnapshotHashes(hashes []common.HashHeight, peer p2p.Peer) error {
	bytM, err := json.Marshal(&snapshotHashesMsg{Hashes: hashes})
	if err != nil {
		return errors.New("SendSnapshotHashes, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.SnapshotHashes, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("SendSnapshotHashes, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err

}

func (ser *sender) RequestAccountHash(address string, height common.HashHeight, prevCnt uint64) error {
	log.Info("fetch account data, account:%s, height:%d, prevCnt:%d, hash:%s.", address, height.Height, prevCnt, height.Hash)
	peer, e := ser.net.BestPeer()
	if e != nil {
		log.Error("sendAccountHash, can't get best peer. err:%v", e)
		return e
	}
	m := requestAccountHashMsg{Address: address, Height: height.Height, Hash: height.Hash, PrevCnt: prevCnt}
	bytM, err := json.Marshal(&m)
	if err != nil {
		return errors.New("sendAccountHash, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestAccountHash, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("sendAccountHash, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (ser *sender) RequestSnapshotHash(height common.HashHeight, prevCnt uint64) error {
	log.Info("fetch snapshot data, height:%d, prevCnt:%d, hash:%s.", height.Height, prevCnt, height.Hash)
	peer, e := ser.net.BestPeer()
	if e != nil {
		log.Error("sendSnapshotHash, can't get best peer. err:%v", e)
		return e
	}
	m := requestSnapshotHashMsg{Height: height.Height, Hash: height.Hash, PrevCnt: prevCnt}
	bytM, err := json.Marshal(&m)
	if err != nil {
		return errors.New("sendSnapshotHash, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestSnapshotHash, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("sendSnapshotHash, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (ser *sender) requestSnapshotBlockByPeer(height common.HashHeight, peer p2p.Peer) error {
	m := requestSnapshotBlockMsg{Hashes: []common.HashHeight{height}}
	bytM, err := json.Marshal(&m)
	if err != nil {
		return errors.New("requestSnapshotBlockByPeer, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestSnapshotBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("requestSnapshotBlockByPeer, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}

func (ser *sender) RequestAccountBlocks(address string, hashes []common.HashHeight) error {
	peer, e := ser.net.BestPeer()
	if e != nil {
		log.Error("RequestAccountBlocks, can't get best peer. err:%v", e)
		return e
	}
	m := requestAccountBlockMsg{Address: address, Hashes: hashes}
	bytM, err := json.Marshal(&m)
	if err != nil {
		return errors.New("RequestAccountBlocks, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestAccountBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("RequestAccountBlocks, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}
func (ser *sender) RequestSnapshotBlocks(hashes []common.HashHeight) error {
	peer, e := ser.net.BestPeer()
	if e != nil {
		log.Error("RequestSnapshotBlocks, can't get best peer. err:%v", e)
		return e
	}
	m := requestSnapshotBlockMsg{Hashes: hashes}
	bytM, err := json.Marshal(&m)
	if err != nil {
		return errors.New("RequestSnapshotBlocks, format fail. err:" + err.Error())
	}
	msg := p2p.NewMsg(common.RequestSnapshotBlocks, bytM)
	err = peer.Write(msg)
	if err != nil {
		log.Error("RequestSnapshotBlocks, write peer fail. peer:%s, err:%v", peer.Id(), err)
	}
	return err
}
