package reader

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/tosih/motronic-m21-tool/pkg/models"
)

// ReadConfigParams reads all configuration parameters from the ECU file
func ReadConfigParams(filename string) (*models.ECUConfig, error) {
	config := &models.ECUConfig{
		Params: models.ConfigParams,
		Values: make(map[string]float64),
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	for _, param := range models.ConfigParams {
		value, err := readConfigValue(f, param)
		if err != nil {
			continue // Skip if error reading
		}
		config.Values[param.Name] = value
	}

	return config, nil
}

func readConfigValue(f *os.File, param models.ConfigParam) (float64, error) {
	_, err := f.Seek(param.Offset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	var rawValue uint64

	switch param.DataType {
	case "uint8":
		var val uint8
		if err := binary.Read(f, binary.LittleEndian, &val); err != nil {
			return 0, err
		}
		rawValue = uint64(val)

	case "uint16":
		var val uint16
		if err := binary.Read(f, binary.LittleEndian, &val); err != nil {
			return 0, err
		}
		rawValue = uint64(val)

	case "int8":
		var val int8
		if err := binary.Read(f, binary.LittleEndian, &val); err != nil {
			return 0, err
		}
		rawValue = uint64(val)

	case "int16":
		var val int16
		if err := binary.Read(f, binary.LittleEndian, &val); err != nil {
			return 0, err
		}
		rawValue = uint64(val)
	}

	// Apply scale and offset
	realValue := float64(rawValue)*param.Scale + param.Offset2

	return realValue, nil
}
