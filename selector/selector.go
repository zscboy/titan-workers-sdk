package selector

import (
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("selector")

type TunInfo struct {
	NodeID string
	URL    string
	Relays []string
	Auth   string
}

type TunSelector interface {
	GetTunInfos(count int) []*TunInfo
}

const TypeAuto = "auto"
const TypeFix = "fix"
