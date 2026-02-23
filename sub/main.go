package main

import (
	"encoding/binary"
	"log"
	"math"

	"github.com/pebbe/zmq4"
)

func main() {
	sub, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Close()

	// Replace with your Pi’s IP
	err = sub.Connect("tcp://192.168.1.50:5555")
	if err != nil {
		log.Fatal(err)
	}

	sub.SetSubscribe("") // subscribe to all messages

	log.Println("Subscriber connected")

	for {
		msg, err := sub.RecvBytes(0)
		if err != nil {
			log.Println("recv error:", err)
			continue
		}

		ts := math.Float64frombits(binary.BigEndian.Uint64(msg[0:8]))
		temp := math.Float32frombits(binary.BigEndian.Uint32(msg[8:12]))
		press := math.Float32frombits(binary.BigEndian.Uint32(msg[12:16]))
		hum := math.Float32frombits(binary.BigEndian.Uint32(msg[16:20]))

		log.Printf("ts=%.3f temp=%.2fC press=%.2fhPa hum=%.2f%%",
			ts, temp, press, hum)
	}
}
