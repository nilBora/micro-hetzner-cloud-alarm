package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Service  []Service `yaml:"services"`
	Task     []Task    `yaml:"tasks"`
	Workflow Workflow  `yaml:"workflow"`
}

type Workflow struct {
	Transitions []Transition `yaml:"transitions"`
}

type Transition struct {
	Name string `yaml:"name"`
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type Service struct {
	Name    string   `yaml:"name"`
	Ident   string   `yaml:"ident"`
	URL     string   `yaml:"url"`
	Token   string   `yaml:"token"`
	Headers []Header `yaml:"headers"`
}

type Task struct {
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"`
	Url            string   `yaml:"url"`
	Method         string   `yaml:"method"`
	Headers        []Header `yaml:"headers"`
	ResponseStruct string   `yaml:"responseStruct"`
	response       []byte   `yaml:"response"`
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
