package main

import (
	"context"
	"fmt"
)

func NewPublisher(ctx context.Context, cfg *Config) (Publisher, error) {
	switch cfg.Backend {
	case "zmq":
		return NewZMQPublisher(ctx, cfg.ZMQ.Endpoint)
	case "nats":
		return NewNATSPublisher(cfg.NATS.URL, cfg.NATS.Subject)
	default:
		return nil, fmt.Errorf("unknown backend: %s", cfg.Backend)
	}
}
