package reader

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

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

// WriteConfigParam writes a single configuration parameter to the ECU file
func WriteConfigParam(filename string, paramName string, realValue float64) error {
	// Find the parameter
	var param *models.ConfigParam
	for i := range models.ConfigParams {
		if models.ConfigParams[i].Name == paramName {
			param = &models.ConfigParams[i]
			break
		}
	}
	if param == nil {
		return fmt.Errorf("parameter not found: %s", paramName)
	}

	// Validate value is within allowed range
	if realValue < param.MinValue || realValue > param.MaxValue {
		return fmt.Errorf("value %.2f out of range [%.2f, %.2f]", realValue, param.MinValue, param.MaxValue)
	}

	// Create backup before modifying
	if err := createBackup(filename); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Convert real value to raw value
	rawValue := (realValue - param.Offset2) / param.Scale

	// Open file for writing
	f, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Seek to parameter offset
	_, err = f.Seek(param.Offset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// Write based on data type
	switch param.DataType {
	case "uint8":
		val := uint8(rawValue)
		if err := binary.Write(f, binary.LittleEndian, val); err != nil {
			return fmt.Errorf("failed to write uint8: %w", err)
		}

	case "uint16":
		val := uint16(rawValue)
		if err := binary.Write(f, binary.LittleEndian, val); err != nil {
			return fmt.Errorf("failed to write uint16: %w", err)
		}

	case "int8":
		val := int8(rawValue)
		if err := binary.Write(f, binary.LittleEndian, val); err != nil {
			return fmt.Errorf("failed to write int8: %w", err)
		}

	case "int16":
		val := int16(rawValue)
		if err := binary.Write(f, binary.LittleEndian, val); err != nil {
			return fmt.Errorf("failed to write int16: %w", err)
		}

	default:
		return fmt.Errorf("unsupported data type: %s", param.DataType)
	}

	return nil
}

// createBackup creates a timestamped backup of the ECU file
func createBackup(filename string) error {
	// Read original file
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.backup_%s", filename, timestamp)

	// Write backup
	return os.WriteFile(backupName, data, 0644)
}
