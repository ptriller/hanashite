package common

import (
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var Version = "dev"

var cfg map[string]interface{}

func Load(path string) {

	reader, err := os.Open(path)
	if err != nil {
		zap.S().Fatalf("failed to read config file %s: %v", path, err)
	}
	defer func(reader *os.File) {
		err := reader.Close()
		if err != nil {
			zap.S().Fatalf("failed to close config file %s: %v", path, err)
		}
	}(reader)

	dec := yaml.NewDecoder(reader)
	err = dec.Decode(&cfg)
	if err != nil {
		zap.S().Fatalf("failed to parse config file %s: %v", path, err)
	}
}

func FetchConfig(key string, v interface{}) error {
	if cfg[key] == nil {
		return errors.New(fmt.Sprintf("Config does not contain key: %s", key))
	}
	dbBytes, err := yaml.Marshal(cfg[key])
	if err != nil {
		return err
	}
	return yaml.Unmarshal(dbBytes, v)
}
