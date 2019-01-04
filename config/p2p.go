package config

type P2P struct {
	Name string `json:"Names""`

	// use for sign data
	PrivateKey string `json:"PeerKey"`

	// `MaxPeers` is the maximum number of peers that can be connected.
	MaxPeers uint `json:"MaxPeers"`

	// `MaxPassivePeersRatio` is the ratio of MaxPeers that initiate an active connection to this node.
	// the actual value is `MaxPeers / MaxPassivePeersRatio`
	MaxPassivePeersRatio uint `json:"MaxPassivePeersRatio"`

	// `MaxPendingPeers` is the maximum number of peers that wait to connect.
	MaxPendingPeers uint `json:"MaxPendingPeers"`

	BootNodes []string `json:"BootNodes"`

	Port uint `json:"Port"`

	Datadir string `json:"Datadir"`

	NetID uint `json:"NetID"`
}
