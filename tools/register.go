package tools

import (
	"web3-service-agent/internal/tool"
	"web3-service-agent/tools/bridge"
	"web3-service-agent/tools/dex"
	"web3-service-agent/tools/market"
	yieldtool "web3-service-agent/tools/yield"
)

func RegisterAll(registry *tool.Registry) error {
	registrars := []func(*tool.Registry) error{
		market.Register,
		dex.Register,
		bridge.Register,
		yieldtool.Register,
	}

	for _, register := range registrars {
		if err := register(registry); err != nil {
			return err
		}
	}
	return nil
}
