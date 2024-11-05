package config

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/NotSoFancyName/producer-consumer/pkg/persistence"
	"github.com/NotSoFancyName/producer-consumer/system/producer/producer"
)

type Config struct {
	LogLevel          string             `yaml:"log_level"`
	PersistenceConfig persistence.Config `yaml:"persistence"`
	ProducerConfig    producer.Config    `yaml:"producer"`
}

func Load(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
