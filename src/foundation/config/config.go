package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LastConnection stores the last database and company used
type LastConnection struct {
	DatabasePath string `json:"database_path"`
	Company      string `json:"company"`
}

const configFileName = ".openerp"

// getConfigPath returns the path to the config file
func getConfigPath() string {
	// Use current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return configFileName
	}
	return filepath.Join(cwd, configFileName)
}

// LoadLastConnection loads the last connection from config file
func LoadLastConnection() (*LastConnection, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err // File doesn't exist or can't be read
	}

	var conn LastConnection
	if err := json.Unmarshal(data, &conn); err != nil {
		return nil, err
	}

	return &conn, nil
}

// SaveLastConnection saves the current connection to config file
func SaveLastConnection(dbPath, company string) error {
	conn := LastConnection{
		DatabasePath: dbPath,
		Company:      company,
	}

	data, err := json.MarshalIndent(conn, "", "  ")
	if err != nil {
		return err
	}

	configPath := getConfigPath()
	return os.WriteFile(configPath, data, 0644)
}

// ClearLastConnection removes the config file
func ClearLastConnection() error {
	configPath := getConfigPath()
	return os.Remove(configPath)
}
