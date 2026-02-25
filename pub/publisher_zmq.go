package main

import (
	"context"

	zmq "github.com/go-zeromq/zmq4"
)

type ZMQPublisher struct {
	pub zmq.Socket
}

func NewZMQPublisher(ctx context.Context, endpoint string) (*ZMQPublisher, error) {
	pub := zmq.NewPub(ctx)
	if err := pub.Listen(endpoint); err != nil {
		return nil, err
	}
	return &ZMQPublisher{pub: pub}, nil
}

func (z *ZMQPublisher) Publish(msg []byte) error {
	return z.pub.Send(zmq.NewMsg(msg))
}

func (z *ZMQPublisher) Close() error {
	return z.pub.Close()
}
