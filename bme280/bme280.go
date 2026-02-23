package bme280

import (
	"encoding/binary"
	"fmt"
	"time"
)

// SPI is implemented by SpiDev and used by the driver.
type SPI interface {
	Transfer(tx []byte) ([]byte, error)
}

type BME280 struct {
	spi SPI

	digT1 uint16
	digT2 int16
	digT3 int16

	digP1 uint16
	digP2 int16
	digP3 int16
	digP4 int16
	digP5 int16
	digP6 int16
	digP7 int16
	digP8 int16
	digP9 int16

	digH1 uint8
	digH2 int16
	digH3 uint8
	digH4 int16
	digH5 int16
	digH6 int8

	tFine int32
}

const (
	regID       = 0xD0
	regReset    = 0xE0
	regCtrlHum  = 0xF2
	regCtrlMeas = 0xF4
	regConfig   = 0xF5
	regPressMSB = 0xF7

	resetCmd = 0xB6
)

// New initializes the BME280 and configures oversampling.
func New(spi SPI) (*BME280, error) {
	b := &BME280{spi: spi}

	id, err := b.readReg(regID)
	if err != nil {
		return nil, err
	}
	if id != 0x60 {
		return nil, fmt.Errorf("unexpected chip id: 0x%02x", id)
	}

	if err := b.writeReg(regReset, resetCmd); err != nil {
		return nil, err
	}
	time.Sleep(5 * time.Millisecond)

	if err := b.readCalibration(); err != nil {
		return nil, err
	}

	// humidity oversampling x1
	if err := b.writeReg(regCtrlHum, 0x01); err != nil {
		return nil, err
	}
	// temp/press oversampling x1, normal mode
	if err := b.writeReg(regCtrlMeas, 0x27); err != nil {
		return nil, err
	}
	// standby 1000ms, filter off
	if err := b.writeReg(regConfig, 0xA0); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *BME280) readReg(reg byte) (byte, error) {
	tx := []byte{reg | 0x80, 0x00}
	rx, err := b.spi.Transfer(tx)
	if err != nil {
		return 0, err
	}
	return rx[1], nil
}

func (b *BME280) writeReg(reg, val byte) error {
	tx := []byte{reg & 0x7F, val}
	_, err := b.spi.Transfer(tx)
	return err
}

func (b *BME280) readBlock(reg byte, n int) ([]byte, error) {
	tx := make([]byte, n+1)
	tx[0] = reg | 0x80
	rx, err := b.spi.Transfer(tx)
	if err != nil {
		return nil, err
	}
	return rx[1:], nil
}

func (b *BME280) readCalibration() error {
	buf1, err := b.readBlock(0x88, 26)
	if err != nil {
		return err
	}
	buf2, err := b.readBlock(0xE1, 7)
	if err != nil {
		return err
	}

	b.digT1 = binary.LittleEndian.Uint16(buf1[0:])
	b.digT2 = int16(binary.LittleEndian.Uint16(buf1[2:]))
	b.digT3 = int16(binary.LittleEndian.Uint16(buf1[4:]))

	b.digP1 = binary.LittleEndian.Uint16(buf1[6:])
	b.digP2 = int16(binary.LittleEndian.Uint16(buf1[8:]))
	b.digP3 = int16(binary.LittleEndian.Uint16(buf1[10:]))
	b.digP4 = int16(binary.LittleEndian.Uint16(buf1[12:]))
	b.digP5 = int16(binary.LittleEndian.Uint16(buf1[14:]))
	b.digP6 = int16(binary.LittleEndian.Uint16(buf1[16:]))
	b.digP7 = int16(binary.LittleEndian.Uint16(buf1[18:]))
	b.digP8 = int16(binary.LittleEndian.Uint16(buf1[20:]))
	b.digP9 = int16(binary.LittleEndian.Uint16(buf1[22:]))

	b.digH1 = buf1[25]
	b.digH2 = int16(binary.LittleEndian.Uint16(buf2[0:]))
	b.digH3 = buf2[2]
	b.digH4 = int16(int16(buf2[3])<<4 | int16(buf2[4]&0x0F))
	b.digH5 = int16(int16(buf2[5])<<4 | int16(buf2[4]>>4))
	b.digH6 = int8(buf2[6])

	return nil
}

// Read returns temperature (°C), pressure (hPa), humidity (%).
func (b *BME280) Read() (float32, float32, float32, error) {
	raw, err := b.readBlock(regPressMSB, 8)
	if err != nil {
		return 0, 0, 0, err
	}

	adcP := int32(raw[0])<<12 | int32(raw[1])<<4 | int32(raw[2])>>4
	adcT := int32(raw[3])<<12 | int32(raw[4])<<4 | int32(raw[5])>>4
	adcH := int32(raw[6])<<8 | int32(raw[7])

	t := b.compensateTemp(adcT)
	p := b.compensatePress(adcP)
	h := b.compensateHum(adcH)

	return t, p, h, nil
}

func (b *BME280) compensateTemp(adcT int32) float32 {
	var1 := (((adcT >> 3) - (int32(b.digT1) << 1)) * int32(b.digT2)) >> 11
	var2 := (((((adcT >> 4) - int32(b.digT1)) * ((adcT >> 4) - int32(b.digT1))) >> 12) * int32(b.digT3)) >> 14
	b.tFine = var1 + var2
	T := (b.tFine*5 + 128) >> 8
	return float32(T) / 100.0
}

func (b *BME280) compensatePress(adcP int32) float32 {
	var1 := int64(b.tFine) - 128000
	var2 := var1 * var1 * int64(b.digP6)
	var2 = var2 + ((var1 * int64(b.digP5)) << 17)
	var2 = var2 + (int64(b.digP4) << 35)
	var1 = ((var1 * var1 * int64(b.digP3)) >> 8) + ((var1 * int64(b.digP2)) << 12)
	var1 = (((int64(1) << 47) + var1) * int64(b.digP1)) >> 33

	if var1 == 0 {
		return 0
	}

	p := int64(1048576 - adcP)
	p = (((p << 31) - var2) * 3125) / var1
	var1 = (int64(b.digP9) * (p >> 13) * (p >> 13)) >> 25
	var2 = (int64(b.digP8) * p) >> 19
	p = ((p + var1 + var2) >> 8) + (int64(b.digP7) << 4)

	return float32(p) / 25600.0
}

func (b *BME280) compensateHum(adcH int32) float32 {
	vx := int32(b.tFine) - 76800
	vx = (((((adcH << 14) - (int32(b.digH4) << 20) - (int32(b.digH5) * vx)) + 16384) >> 15) *
		(((((((vx*int32(b.digH6))>>10)*(((vx*int32(b.digH3))>>11)+32768))>>10)+2097152)*int32(b.digH2) + 8192) >> 14))
	vx = vx - (((vx>>15)*(vx>>15))>>7)*int32(b.digH1)>>4
	if vx < 0 {
		vx = 0
	}
	if vx > 419430400 {
		vx = 419430400
	}
	h := float32(vx>>12) / 1024.0
	return h
}
