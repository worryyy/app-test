package topic

const merchantPowerBit = 2

func isMerchantPower(power int) bool {
	return ((power >> merchantPowerBit) & 1) == 1
}
