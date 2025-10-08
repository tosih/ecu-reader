package reader

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/tosih/motronic-m21-tool/pkg/models"
)

// ReadMap reads a map from the binary file at the specified configuration
func ReadMap(filename string, cfg models.MapConfig) (*models.ECUMap, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = f.Seek(cfg.Offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	data := make([][]float64, cfg.Rows)
	for i := 0; i < cfg.Rows; i++ {
		data[i] = make([]float64, cfg.Cols)
		for j := 0; j < cfg.Cols; j++ {
			var value float64

			if cfg.DataType == "uint8" {
				var rawValue uint8
				err := binary.Read(f, binary.LittleEndian, &rawValue)
				if err != nil {
					return nil, err
				}
				value = float64(rawValue)*cfg.Scale + cfg.Offset2
			} else {
				var rawValue uint16
				err := binary.Read(f, binary.LittleEndian, &rawValue)
				if err != nil {
					return nil, err
				}
				value = float64(rawValue)*cfg.Scale + cfg.Offset2
			}

			data[i][j] = value
		}
	}

	return &models.ECUMap{
		Config: cfg,
		Data:   data,
	}, nil
}

// FindMinMax finds the minimum and maximum values in map data
func FindMinMax(data [][]float64) (float64, float64) {
	min := data[0][0]
	max := data[0][0]

	for _, row := range data {
		for _, val := range row {
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
		}
	}

	return min, max
}

// ReadConfigParam reads a configuration parameter value from the ECU file
func ReadConfigParam(filename string, param models.ConfigParam) (float64, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	_, err = f.Seek(param.Offset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	var value float64
	switch param.DataType {
	case "uint8":
		var rawValue uint8
		err = binary.Read(f, binary.LittleEndian, &rawValue)
		if err != nil {
			return 0, err
		}
		value = float64(rawValue)*param.Scale + param.Offset2

	case "uint16":
		var rawValue uint16
		err = binary.Read(f, binary.LittleEndian, &rawValue)
		if err != nil {
			return 0, err
		}
		value = float64(rawValue)*param.Scale + param.Offset2

	case "int8":
		var rawValue int8
		err = binary.Read(f, binary.LittleEndian, &rawValue)
		if err != nil {
			return 0, err
		}
		value = float64(rawValue)*param.Scale + param.Offset2

	case "int16":
		var rawValue int16
		err = binary.Read(f, binary.LittleEndian, &rawValue)
		if err != nil {
			return 0, err
		}
		value = float64(rawValue)*param.Scale + param.Offset2

	default:
		return 0, nil
	}

	return value, nil
}
