package selector

type FixSelector struct {
	tunInfos []*TunInfo
}

func NewFixSelector(tunInfos []*TunInfo) *FixSelector {
	return &FixSelector{tunInfos: tunInfos}
}

func (fs *FixSelector) GetTunInfos(count int) []*TunInfo {
	return fs.tunInfos
}
