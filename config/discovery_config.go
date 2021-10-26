package config

import (
	"github.com/netsys-lab/dht"
)

type PeerDiscoveryConfig struct {
	EnableDht     bool // start dht node
	DhtPort       uint16
	EnableTracker bool // TODO: implementation currently doesnt support SCION-trackers
	DhtNodes      []dht.Addr
}

// DefaultPeerDisoveryConfig use all supported dynamic peer discovery techniques
func DefaultPeerDisoveryConfig() PeerDiscoveryConfig {
	return PeerDiscoveryConfig{
		EnableDht:     true,
		EnableTracker: false,
		DhtPort:       7000,
	}
}
