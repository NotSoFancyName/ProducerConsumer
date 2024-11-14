package config

import (
	"github.com/NotSoFancyName/producer-consumer/system/consumer/client"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/NotSoFancyName/producer-consumer/pkg/persistence"
	"github.com/NotSoFancyName/producer-consumer/system/consumer/processor"
)

type Config struct {
	LogLevel          string                   `yaml:"log_level"`
	PersistenceConfig persistence.Config       `yaml:"persistence"`
	ProcessorConfig   processor.Config         `yaml:"processor"`
	RateLimiterConfig client.RateLimiterConfig `yaml:"rate_limiter_config"`
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
