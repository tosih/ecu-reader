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
var ConfigParams = []ConfigParam{
	{
		Name:        "Rev Limiter",
		Offset:      0x7000,
		DataType:    "uint8",
		Scale:       50.0,
		Offset2:     0,
		Unit:        "RPM",
		Description: "Maximum engine RPM limit",
		MinValue:    3000,
		MaxValue:    8000,
	},
	{
		Name:        "Idle Speed Target",
		Offset:      0x7001,
		DataType:    "uint8",
		Scale:       10.0,
		Offset2:     0,
		Unit:        "RPM",
		Description: "Target idle speed",
		MinValue:    600,
		MaxValue:    1200,
	},
	{
		Name:        "Fuel Cut RPM",
		Offset:      0x7B40,
		DataType:    "uint8",
		Scale:       50.0,
		Offset2:     0,
		Unit:        "RPM",
		Description: "RPM for overrun fuel cutoff",
		MinValue:    1000,
		MaxValue:    2500,
	},
	{
		Name:        "Fuel Resume RPM",
		Offset:      0x7B41,
		DataType:    "uint8",
		Scale:       50.0,
		Offset2:     0,
		Unit:        "RPM",
		Description: "RPM for fuel resume after cutoff",
		MinValue:    800,
		MaxValue:    2000,
	},
	{
		Name:        "Coolant Temp Enrichment",
		Offset:      0x7A40,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Coolant temperature enrichment multiplier",
		MinValue:    0,
		MaxValue:    2.0,
	},
	{
		Name:        "Air Temp Enrichment",
		Offset:      0x7A41,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Air temperature enrichment multiplier",
		MinValue:    0,
		MaxValue:    2.0,
	},
	{
		Name:        "Throttle Opening Rate",
		Offset:      0x7980,
		DataType:    "uint8",
		Scale:       1.0,
		Offset2:     0,
		Unit:        "%/s",
		Description: "Maximum throttle opening rate",
		MinValue:    10,
		MaxValue:    100,
	},
	{
		Name:        "Boost Limit",
		Offset:      0x7940,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "bar",
		Description: "Maximum boost pressure limit",
		MinValue:    0,
		MaxValue:    2.5,
	},
}
