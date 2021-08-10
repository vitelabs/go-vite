package upgrade

import "fmt"

func CheckUpgradeCfg(points map[string]*UpgradePoint) error {
	for k, point := range points {
		if point == nil {
			return fmt.Errorf("the fork point %s can't be nil", k)
		}

		if point.Height <= 0 {
			return fmt.Errorf("the height of fork point %s is 0", k)
		}

		if point.Version <= 0 {
			return fmt.Errorf("the version of fork point %s is 0", k)
		}
	}
	return nil
}
