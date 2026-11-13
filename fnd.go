package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"os"
	"sync"
	"time"

	"github.com/suapapa/go_devices/tm1638"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

const (
	fndBlank      = "        "
	fndLen        = 8
	pinData       = "17"
	pinClock      = "27"
	pinStrobe     = "22"
	spinStepCount = 100
	spinStepDelay = 30 * time.Millisecond
)

// FND manages the 8-digit TM1638 7-segment display.
type FND struct {
	m          *tm1638.Module
	lastString string

	sync.Mutex
}

// NewFND initializes the TM1638 display, or returns a log-only display in dry-run mode.
func NewFND(dryRun bool) *FND {
	if dryRun {
		return &FND{}
	}

	if _, err := host.Init(); err != nil {
		log.Fatalf("failed to init host GPIO: %v", err)
	}

	dataPin := gpioreg.ByName(pinData)
	clkPin := gpioreg.ByName(pinClock)
	stbPin := gpioreg.ByName(pinStrobe)
	if dataPin == nil || clkPin == nil || stbPin == nil {
		log.Fatal("tm1638 GPIO pins not available (use -n for dry run)")
	}

	m, err := tm1638.Open(dataPin, clkPin, stbPin)
	if err != nil {
		log.Fatalf("failed to open tm1638: %v", err)
	}
	return &FND{
		m: m,
	}
}

// SetString displays up to 8 characters on the FND, padding shorter strings with spaces.
func (f *FND) SetString(s string) {
	f.Lock()
	defer f.Unlock()

	if len(s) > fndLen {
		s = s[:fndLen]
	} else if len(s) < fndLen {
		s += fndBlank[:fndLen-len(s)]
	}

	if f.lastString == s {
		return
	}

	if f.m != nil {
		f.m.SetString(s)
	} else {
		log.Printf("dry run fnd: %q", s)
	}
	f.lastString = s
}

// Welcome animates random characters until a local IPv4 address is resolved, then shows the suffix.
func (f *FND) Welcome() error {
	f.Lock()
	defer f.Unlock()

	if f.m == nil {
		log.Println("dry run fnd: welcome")
		return nil
	}

	for {
		for range spinStepCount {
			f.m.SetString(randString())
			time.Sleep(spinStepDelay)
		}

		_, ip, _, err := resolveNet()
		if err != nil || ip == "" {
			continue
		}

		suffix := ip
		if len(suffix) > 3 {
			suffix = suffix[len(suffix)-3:]
		}
		f.m.SetString("srmt-" + suffix)
		return nil
	}
}

func randString() string {
	const (
		minASCII = 32
		maxASCII = 126
	)
	buf := make([]byte, fndLen)
	for i := range buf {
		buf[i] = byte(minASCII + rand.IntN(maxASCII-minASCII))
	}
	return string(buf)
}

// resolveNet returns hostname, IPv4, MAC, and error for the first active non-loopback interface.
func resolveNet() (string, string, string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", "", "", err
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", "", err
	}

	for _, i := range ifaces {
		isDown := (i.Flags & net.FlagUp) == 0
		isLoopback := (i.Flags & net.FlagLoopback) != 0
		if isDown || isLoopback {
			continue
		}

		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := extractIPv4(addr)
			if ip != nil {
				return hostname, ip.String(), i.HardwareAddr.String(), nil
			}
		}
	}
	return "", "", "", fmt.Errorf("cannot resolve the IP")
}

func extractIPv4(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil {
		return nil
	}
	return ip.To4()
}
