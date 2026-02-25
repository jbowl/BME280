package main

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Backend string `yaml:"backend"` // "zmq" or "nats"

	ZMQ struct {
		Endpoint string `yaml:"endpoint"` // e.g. tcp://127.0.0.1:5555
	} `yaml:"zmq"`

	NATS struct {
		URL     string `yaml:"url"`     // e.g. nats://127.0.0.1:4222
		Subject string `yaml:"subject"` // e.g. bme280.readings
	} `yaml:"nats"`

	SPI struct {
		Device string `yaml:"device"` // e.g. /dev/spidev0.0
	} `yaml:"spi"`

	Metrics struct {
		Addr string `yaml:"addr"` // e.g. :9100
	} `yaml:"metrics"`

	Publish struct {
		Interval time.Duration `yaml:"interval"` // e.g. 1s
	} `yaml:"publish"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	if cfg.Backend == "" {
		cfg.Backend = "zmq"
	}
	if cfg.SPI.Device == "" {
		cfg.SPI.Device = "/dev/spidev0.0"
	}
	if cfg.Metrics.Addr == "" {
		cfg.Metrics.Addr = ":9100"
	}
	if cfg.Publish.Interval == 0 {
		cfg.Publish.Interval = time.Second
	}
	if cfg.ZMQ.Endpoint == "" {
		cfg.ZMQ.Endpoint = "tcp://127.0.0.1:5555"
	}
	if cfg.NATS.URL == "" {
		cfg.NATS.URL = "nats://127.0.0.1:4222"
	}
	if cfg.NATS.Subject == "" {
		cfg.NATS.Subject = "bme280.readings"
	}

	return &cfg, nil
}
