package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	zmq "github.com/go-zeromq/zmq4"
)

func RunZMQSubscriber(ctx context.Context, endpoint string) error {
	sub := zmq.NewSub(ctx)
	defer sub.Close()

	if err := sub.Dial(endpoint); err != nil {
		return err
	}
	if err := sub.SetOption(zmq.OptionSubscribe, ""); err != nil {
		return err
	}

	for {
		msg, err := sub.Recv()
		if err != nil {
			return err
		}
		if len(msg.Frames) == 0 {
			continue
		}
		buf := msg.Frames[0]

		if len(buf) != 20 {
			fmt.Println("invalid length:", len(buf))
			continue
		}

		ts := math.Float64frombits(binary.BigEndian.Uint64(buf[0:8]))
		t := math.Float32frombits(binary.BigEndian.Uint32(buf[8:12]))
		p := math.Float32frombits(binary.BigEndian.Uint32(buf[12:16]))
		h := math.Float32frombits(binary.BigEndian.Uint32(buf[16:20]))

		fmt.Printf("%s ts=%.3f temp=%.2fC press=%.2fhPa hum=%.2f%%\n",
			time.Now().Format(time.RFC3339), ts, t, p, h)
	}
}
