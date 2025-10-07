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
// Maps 0-2 are CONFIRMED via binary scan analysis
// Maps 3+ are high-confidence candidates from scan results
var MapConfigs = []MapConfig{
	// CONFIRMED MAPS (validated in both binary files)
	{
		Name:        "Main Fuel Map",
		Offset:      0x6700,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.04,
		Offset2:     0,
		Unit:        "ms",
		Description: "Primary fuel injection duration map (CONFIRMED)",
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
		Description: "Spark advance timing map (CONFIRMED)",
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
		Description: "Target air-fuel ratio map (CONFIRMED)",
	},

	// HIGH-CONFIDENCE CANDIDATES (from scan analysis)
	{
		Name:        "Correction Table 1",
		Offset:      0x60C0,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Limits/correction table (variance: 100.3)",
	},
	{
		Name:        "Fuel/Timing Trim 1",
		Offset:      0x6CC0,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Fuel or timing trim table (variance: 260.9)",
	},
	{
		Name:        "Correction Table 2",
		Offset:      0x6D00,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Correction table (variance: 125.1)",
	},
	{
		Name:        "Fuel/Timing Trim 2",
		Offset:      0x6EC0,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Fuel or timing trim table (variance: 385.8)",
	},
	{
		Name:        "Correction Table 3",
		Offset:      0x6F80,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Correction table (variance: 136.3)",
	},
	{
		Name:        "Trim Table 1",
		Offset:      0x7140,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Trim table (variance: 196.6)",
	},
	{
		Name:        "Trim Table 2",
		Offset:      0x7200,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0,
		Unit:        "%",
		Description: "Trim table (variance: 237.1)",
	},
}
