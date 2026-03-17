package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config представляет полную конфигурацию сервера базы данных.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Storage  StorageConfig  `yaml:"storage"`
	Transaction TransactionConfig `yaml:"transaction"`
}

// ServerConfig содержит настройки сетевого интерфейса.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// StorageConfig содержит параметры подсистемы хранения.
type StorageConfig struct {
	DataDir      string `yaml:"data_dir"`
	WALDir       string `yaml:"wal_dir"`
	MaxMemTableSize int64 `yaml:"max_memtable_size_bytes"`
	FlushThreshold int   `yaml:"flush_threshold"`
}

// TransactionConfig содержит настройки управления транзакциями.
type TransactionConfig struct {
	IsolationLevel string `yaml:"isolation_level"`
	MaxLockWaitTimeMs int `yaml:"max_lock_wait_time_ms"`
}

// LoadConfig загружает и парсит конфигурационный файл.
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла конфигурации: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("ошибка парсинга YAML конфигурации: %w", err)
	}

	// Валидация обязательных полей
	if config.Server.Port == 0 {
		config.Server.Port = 5432 // Значение по умолчанию для совместимости
	}
	if config.Storage.DataDir == "" {
		config.Storage.DataDir = "./data"
	}

	return &config, nil
}