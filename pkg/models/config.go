package models

// ConfigParam defines a single configuration parameter in the ECU
type ConfigParam struct {
	Name        string
	Offset      int64
	DataType    string // uint8, uint16, int8, int16
	Scale       float64
	Offset2     float64
	Unit        string
	Description string
	MinValue    float64
	MaxValue    float64
}

// ECUConfig holds all configuration parameters
type ECUConfig struct {
	Params []ConfigParam
	Values map[string]float64
}

// Common Motronic M2.1 configuration parameters
// Values verified against Porsche 964 binary (964618124-03_1267357006.BIN)
var ConfigParams = []ConfigParam{
	{
		Name:        "Rev Limiter",
		Offset:      0x7000,
		DataType:    "uint8",
		Scale:       85.37,
		Offset2:     0,
		Unit:        "RPM",
		Description: "Maximum engine RPM limit (Stock 964: ~7000 RPM, safe max: 6800 RPM)",
		MinValue:    6000,
		MaxValue:    7500,
	},
	{
		Name:        "Idle Speed Target",
		Offset:      0x7001,
		DataType:    "uint8",
		Scale:       10.0,
		Offset2:     0,
		Unit:        "RPM",
		Description: "Target idle speed (Stock 964: ~820 RPM)",
		MinValue:    650,
		MaxValue:    1100,
	},
	{
		Name:        "Unknown Param 1",
		Offset:      0x7002,
		DataType:    "uint8",
		Scale:       1.0,
		Offset2:     0,
		Unit:        "raw",
		Description: "Unknown parameter at 0x7002 (Stock 964: 75)",
		MinValue:    0,
		MaxValue:    255,
	},
	{
		Name:        "Unknown Param 2",
		Offset:      0x7003,
		DataType:    "uint8",
		Scale:       1.0,
		Offset2:     0,
		Unit:        "raw",
		Description: "Unknown parameter at 0x7003 (Stock 964: 70)",
		MinValue:    0,
		MaxValue:    255,
	},
}
