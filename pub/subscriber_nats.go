package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/nats-io/nats.go"
)

func RunNATSSubscriber(url, subject string) error {
	nc, err := nats.Connect(url)
	if err != nil {
		return err
	}
	defer nc.Close()

	_, err = nc.Subscribe(subject, func(m *nats.Msg) {
		buf := m.Data
		if len(buf) != 20 {
			fmt.Println("invalid length:", len(buf))
			return
		}

		ts := math.Float64frombits(binary.BigEndian.Uint64(buf[0:8]))
		t := math.Float32frombits(binary.BigEndian.Uint32(buf[8:12]))
		p := math.Float32frombits(binary.BigEndian.Uint32(buf[12:16]))
		h := math.Float32frombits(binary.BigEndian.Uint32(buf[16:20]))

		fmt.Printf("%s ts=%.3f temp=%.2fC press=%.2fhPa hum=%.2f%%\n",
			time.Now().Format(time.RFC3339), ts, t, p, h)
	})
	if err != nil {
		return err
	}

	select {}
}
