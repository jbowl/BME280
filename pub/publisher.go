package main

type Publisher interface {
	Publish([]byte) error
	Close() error
}
