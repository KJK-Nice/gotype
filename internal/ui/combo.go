package ui

import "fmt"

const (
	comboHUDMin  = 10
	comboHotMin  = 25
	comboFireMin = 50
	chainHUDMin  = 2
)

func comboHUD(combo int) string {
	if combo < comboHUDMin {
		return ""
	}
	return fmt.Sprintf("×%d", combo)
}

func chainHUD(chain int) string {
	if chain < chainHUDMin {
		return ""
	}
	return fmt.Sprintf("%d chain", chain)
}

func comboHUDHot(combo int) bool {
	return combo >= comboHotMin
}

func comboHUDFire(combo int) bool {
	return combo >= comboFireMin
}
