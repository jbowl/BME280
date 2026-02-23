package bme280

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	spiIOCMessage0 = 0x40206b00 // SPI_IOC_MESSAGE(1)
)

type spiIocTransfer struct {
	TxBuf       uint64
	RxBuf       uint64
	Len         uint32
	SpeedHz     uint32
	DelayUsecs  uint16
	BitsPerWord uint8
	CsChange    uint8
	TxNBits     uint8
	RxNBits     uint8
	Pad         uint16
}

type SpiDev struct {
	f *os.File
}

// OpenSpiDev opens a Linux spidev device like /dev/spidev0.0.
func OpenSpiDev(path string) (*SpiDev, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &SpiDev{f: f}, nil
}

func (s *SpiDev) Close() error {
	return s.f.Close()
}

// Transfer sends tx and returns rx of the same length.
func (s *SpiDev) Transfer(tx []byte) ([]byte, error) {
	rx := make([]byte, len(tx))
	tr := spiIocTransfer{
		TxBuf:       uint64(uintptr(unsafe.Pointer(&tx[0]))),
		RxBuf:       uint64(uintptr(unsafe.Pointer(&rx[0]))),
		Len:         uint32(len(tx)),
		SpeedHz:     1000000,
		BitsPerWord: 8,
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		s.f.Fd(),
		spiIOCMessage0,
		uintptr(unsafe.Pointer(&tr)),
	)
	if errno != 0 {
		return nil, errno
	}
	return rx, nil
}
