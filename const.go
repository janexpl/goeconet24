package goeconet24

type CommandType uint8

const (
	NewParamKey CommandType = iota
	NewParamIndex
	NewParamName
)

type BoilerStatus uint32

const (
	TurnedOff BoilerStatus = iota
	FireUp1
	FireUp2
	Work
	Supervision
	Halted
	Stop
	BurningOff
	Manual
	Alarm
	Unsealing
	Chimney
	Stabilization
	NoTransmission
)

const HuwTemp = 1281
const COTemp = 1280
const HUWHeater = 59
