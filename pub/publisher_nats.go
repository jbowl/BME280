package main

import "github.com/nats-io/nats.go"

type NATSPublisher struct {
	nc   *nats.Conn
	subj string
}

func NewNATSPublisher(url, subject string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NATSPublisher{nc: nc, subj: subject}, nil
}

func (n *NATSPublisher) Publish(msg []byte) error {
	return n.nc.Publish(n.subj, msg)
}

func (n *NATSPublisher) Close() error {
	n.nc.Flush()
	n.nc.Close()
	return nil
}
