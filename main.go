package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tarm/serial"
)

const (
	defaultSerialPort = "/dev/ttyUSB0"
	defaultStorePath  = "store.json"
	serialBaudRate    = 9600
	serialReadTimeout = 1 * time.Second
	releaseTimeout    = 250 * time.Millisecond
	longPressDelay    = 400 * time.Millisecond
	longPressMinCount = 3
	buttonBufferSize  = 16
	fndShowDuration   = 1 * time.Second
)

type irEvent struct {
	code   uint32
	repeat bool
}

type cliOptions struct {
	port      string
	tvHost    string
	storePath string
	dryRun    bool
}

func main() {
	opts := parseFlags()

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fnd := NewFND(opts.dryRun)
	if err := fnd.Welcome(); err != nil {
		log.Printf("warning: welcome banner failed: %v", err)
	}

	tv := NewTV(opts.tvHost, opts.storePath, opts.dryRun)
	defer tv.Close()

	ser, err := serial.OpenPort(&serial.Config{
		Name:        opts.port,
		Baud:        serialBaudRate,
		ReadTimeout: serialReadTimeout,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer ser.Close()

	codeInC := make(chan irEvent, buttonBufferSize)
	buttonC := make(chan button, buttonBufferSize)

	go detectPresses(rootCtx, codeInC, buttonC)
	go handleButtons(
		rootCtx,
		buttonC,
		fnd,
		tv,
	)

	go func() {
		<-rootCtx.Done()
		_ = ser.Close()
	}()

	scanIRCodes(
		rootCtx,
		ser,
		codeInC,
	)
}

func parseFlags() cliOptions {
	var opts cliOptions
	flag.StringVar(&opts.port, "port", defaultSerialPort, "serial port")
	flag.StringVar(&opts.tvHost, "tv", "", "TV IP or hostname override (auto-discovered if empty)")
	flag.StringVar(&opts.storePath, "store", defaultStorePath, "path to TV pairing store JSON file")
	flag.BoolVar(&opts.dryRun, "n", false, "dry run without GPIO or WebOS TV network calls")
	flag.Parse()
	return opts
}

type pressState struct {
	code      uint32
	count     int
	startTime time.Time
	longFired bool
}

func (p *pressState) onEvent(ev irEvent, now time.Time) (out []button) {
	if ev.repeat {
		if p.code == 0 || ev.code != p.code {
			return nil
		}
		p.count++
		readyForLong := !p.longFired && p.count >= longPressMinCount && now.Sub(p.startTime) >= longPressDelay
		if readyForLong {
			p.longFired = true
			log.Printf("code 0x%x longPress", p.code)
			return []button{codeToButton(p.code, true)}
		}
		return nil
	}

	if p.code != 0 && !p.longFired {
		log.Printf("code 0x%x shortPress", p.code)
		out = append(out, codeToButton(p.code, false))
	}

	p.code = ev.code
	p.count = 1
	p.startTime = now
	p.longFired = false
	return out
}

func (p *pressState) onReleaseTimeout() (button, bool) {
	if p.code == 0 {
		return UNKNOWN, false
	}

	code := p.code
	longFired := p.longFired
	*p = pressState{}

	if longFired {
		return UNKNOWN, false
	}
	log.Printf("code 0x%x shortPress", code)
	return codeToButton(code, false), true
}

func detectPresses(
	ctx context.Context,
	codeC <-chan irEvent,
	buttonC chan<- button,
) {
	defer close(buttonC)

	var state pressState
	releaseTimer := time.NewTimer(releaseTimeout)
	releaseTimer.Stop()
	defer releaseTimer.Stop()

	emit := func(btn button) bool {
		select {
		case buttonC <- btn:
			return true
		case <-ctx.Done():
			return false
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-codeC:
			if !ok {
				return
			}
			for _, btn := range state.onEvent(ev, time.Now()) {
				if !emit(btn) {
					return
				}
			}
			if state.code != 0 {
				releaseTimer.Reset(releaseTimeout)
			}
		case <-releaseTimer.C:
			if btn, ok := state.onReleaseTimeout(); ok {
				if !emit(btn) {
					return
				}
			}
		}
	}
}

func handleButtons(
	ctx context.Context,
	buttonC <-chan button,
	fnd *FND,
	tv *TV,
) {
	var blankGen atomic.Uint64

	for btn := range buttonC {
		log.Printf("button: %s", btn)
		seq := blankGen.Add(1)
		fnd.SetString(btn.String())
		time.AfterFunc(fndShowDuration, func() {
			if ctx.Err() == nil && blankGen.Load() == seq {
				fnd.SetString(fndBlank)
			}
		})

		if err := tv.Handle(ctx, btn); err != nil {
			log.Printf("error doing %s: %v", btn, err)
		}
	}
}

func scanIRCodes(
	ctx context.Context,
	ser io.Reader,
	codeInC chan<- irEvent,
) {
	reader := bufio.NewReader(ser)
	var lastCode uint32
	var partial string

	for ctx.Err() == nil {
		line, err := reader.ReadString('\n')
		if line != "" {
			partial += line
			if strings.HasSuffix(line, "\n") {
				codeStr := strings.TrimSpace(partial)
				partial = ""
				if codeStr != "" {
					code, isRepeat, parseErr := parseIRCode(codeStr, lastCode)
					if parseErr != nil {
						log.Printf("error parsing %q: %v", codeStr, parseErr)
					} else {
						if !isRepeat {
							lastCode = code
						}
						if code != 0 {
							select {
							case codeInC <- irEvent{code: code, repeat: isRepeat}:
							case <-ctx.Done():
								return
							}
						}
					}
				}
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			if ctx.Err() == nil {
				log.Printf("error scanning serial port: %v", err)
			}
			return
		}
	}
}

func parseIRCode(codeStr string, lastCode uint32) (uint32, bool, error) {
	trimmed := strings.TrimSpace(codeStr)
	if trimmed == "0" {
		return lastCode, true, nil
	}

	codeBytes, err := hex.DecodeString(trimmed)
	if err != nil {
		return 0, false, fmt.Errorf("decode hex %q: %w", trimmed, err)
	}
	if len(codeBytes) != 4 {
		return 0, false, fmt.Errorf("unexpected code length: %d", len(codeBytes))
	}

	return binary.BigEndian.Uint32(codeBytes), false, nil
}

func codeToButton(code uint32, longPress bool) button {
	log.Printf("code: 0x%x", code)

	codeMap := shortPressButton
	if longPress {
		codeMap = longPressButton
	}

	btn, ok := codeMap[code]
	if !ok {
		log.Printf("unknown code: 0x%x", code)
		return UNKNOWN
	}
	return btn
}
