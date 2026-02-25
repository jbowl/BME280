package main

import (
	"context"
	"encoding/binary"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jbowl/bme280"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	tempGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bme280_temperature_celsius",
		Help: "Temperature from BME280",
	})
	pressGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bme280_pressure_hpa",
		Help: "Pressure from BME280",
	})
	humGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bme280_humidity_percent",
		Help: "Humidity from BME280",
	})
)

func init() {
	prometheus.MustRegister(tempGauge, pressGauge, humGauge)
}

func startMetricsServer(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Error().Err(err).Msg("metrics server failed")
		}
	}()
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("load config failed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigCh
		log.Info().Str("signal", s.String()).Msg("shutdown requested")
		cancel()
	}()

	spiDev, err := bme280.OpenSpiDev(cfg.SPI.Device)
	if err != nil {
		log.Fatal().Err(err).Msg("open spidev failed")
	}
	defer spiDev.Close()

	sensor, err := bme280.New(spiDev)
	if err != nil {
		log.Fatal().Err(err).Msg("init bme280 failed")
	}

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("publisher init failed")
	}
	defer pub.Close()

	startMetricsServer(cfg.Metrics.Addr)

	daemon.SdNotify(false, daemon.SdNotifyReady)

	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				daemon.SdNotify(false, daemon.SdNotifyWatchdog)
			}
		}
	}()

	log.Info().Str("backend", cfg.Backend).Msg("bme280 publisher started")

	ticker := time.NewTicker(cfg.Publish.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("shutting down publisher")
			return
		case <-ticker.C:
			t, p, h, err := sensor.Read()
			if err != nil {
				log.Error().Err(err).Msg("sensor read failed")
				continue
			}

			tempGauge.Set(float64(t))
			pressGauge.Set(float64(p))
			humGauge.Set(float64(h))

			buf := make([]byte, 20)
			ts := float64(time.Now().UnixNano()) / 1e9
			binary.BigEndian.PutUint64(buf[0:], math.Float64bits(ts))
			binary.BigEndian.PutUint32(buf[8:], math.Float32bits(t))
			binary.BigEndian.PutUint32(buf[12:], math.Float32bits(p))
			binary.BigEndian.PutUint32(buf[16:], math.Float32bits(h))

			if err := pub.Publish(buf); err != nil {
				log.Error().Err(err).Msg("publish failed")
			} else {
				log.Debug().
					Float32("temp", t).
					Float32("press", p).
					Float32("hum", h).
					Msg("published")
			}
		}
	}
}
