package models

// MapConfig defines the structure of a map in the ECU file
type MapConfig struct {
	Name        string
	Offset      int64
	Rows        int
	Cols        int
	DataType    string
	Scale       float64
	Offset2     float64
	Unit        string
	Description string
}

// ECUMap represents a 2D map from the ECU
type ECUMap struct {
	Config MapConfig
	Data   [][]float64
}

// Predefined map configurations for Motronic M2.1
var MapConfigs = []MapConfig{
	{
		Name:        "Main Fuel Map",
		Offset:      0x6700,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.04,
		Offset2:     0,
		Unit:        "ms",
		Description: "Primary fuel injection duration map",
	},
	{
		Name:        "Ignition Timing Map",
		Offset:      0x6780,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.75,
		Offset2:     -24.0,
		Unit:        "deg",
		Description: "Spark advance timing map",
	},
	{
		Name:        "Lambda Target Map",
		Offset:      0x6800,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0.5,
		Unit:        "λ",
		Description: "Target air-fuel ratio map",
	},
	{
		Name:        "Boost Control Map",
		Offset:      0x7900,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       0.1,
		Offset2:     0,
		Unit:        "bar",
		Description: "Wastegate duty cycle / boost target",
	},
	{
		Name:        "Cold Start Enrichment",
		Offset:      0x7A00,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       0.02,
		Offset2:     0,
		Unit:        "%",
		Description: "Cold start fuel enrichment multiplier",
	},
	{
		Name:        "Fuel Enrichment Map",
		Offset:      0x6880,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Additional fuel enrichment percentage",
	},
	{
		Name:        "Idle Control Map",
		Offset:      0x7800,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       0.5,
		Offset2:     0,
		Unit:        "steps",
		Description: "Idle air control valve position",
	},
	{
		Name:        "Air Temp Correction",
		Offset:      0x6900,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Intake air temperature correction factor",
	},
	{
		Name:        "Throttle Position Map",
		Offset:      0x6980,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.4,
		Offset2:     0,
		Unit:        "%",
		Description: "Throttle position vs load correlation",
	},
	{
		Name:        "Overrun Fuel Cut",
		Offset:      0x7B00,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       1.0,
		Offset2:     0,
		Unit:        "rpm/10",
		Description: "Deceleration fuel cutoff thresholds",
	},
}
