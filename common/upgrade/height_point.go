package upgrade

type HeightPoint interface {
	IsVersion12Upgrade() bool
	IsDexFeeUpgrade() bool
}

type heightPoint struct {
	height uint64
}

func NewHeightPoint(height uint64) HeightPoint {
	return &heightPoint{height}
}

func (p heightPoint) IsDexFeeUpgrade() bool {
	return IsDexFeeUpgrade(p.height)
}

func (p heightPoint) IsVersion12Upgrade() bool {
	return IsVersion12Upgrade(p.height)
}

type mockHeightPoint struct {
	box    UpgradeBox
	height uint64
}

// IsDexFeeUpgrade implements HeightPoint
func (m mockHeightPoint) IsDexFeeUpgrade() bool {
	return m.box.isActive(3, m.height)
}

// IsVersion12Upgrade implements HeightPoint
func (m mockHeightPoint) IsVersion12Upgrade() bool {
	return m.box.isActive(12, m.height)
}

func NewMockPoint(height uint64, box UpgradeBox) HeightPoint {
	return &mockHeightPoint{box, height}
}
