package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Service []Service `yaml:"services"`
}

type Service struct {
	Name    string   `yaml:"name"`
	Ident   string   `yaml:"ident"`
	URL     string   `yaml:"url"`
	Token   string   `yaml:"token"`
	Headers []Header `yaml:"headers"`
}

type Header struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func LoadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
