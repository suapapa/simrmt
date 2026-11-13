package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tarm/serial"
)

const (
	defaultSerialPort = "/dev/ttyUSB0"
	defaultStorePath  = "store.json"
	serialBaudRate    = 9600
	serialReadTimeout = 1 * time.Second
	irTickInterval    = 100 * time.Millisecond
	shortPressDelay   = 100 * time.Millisecond
	longPressDelay    = 500 * time.Millisecond
	longPressMinCount = 3
)

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

	codeInC := make(chan uint32)
	buttonC := make(chan button)

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

func detectPresses(
	ctx context.Context,
	codeC <-chan uint32,
	buttonC chan<- button,
) {
	defer close(buttonC)

	var repeatedCode uint32
	var repeatCnt int
	var downTime time.Time

	ticker := time.NewTicker(irTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case code := <-codeC:
			if repeatedCode == code {
				repeatCnt++
			} else {
				repeatedCode = code
				repeatCnt = 1
			}
			downTime = time.Now()
		case <-ticker.C:
			if repeatedCode == 0 {
				continue
			}

			elapsed := time.Since(downTime)
			isLongPress := elapsed > longPressDelay && repeatCnt >= longPressMinCount
			isShortPress := elapsed > shortPressDelay && repeatCnt < longPressMinCount
			if !isLongPress && !isShortPress {
				continue
			}

			if isLongPress {
				log.Printf("code 0x%x longPress", repeatedCode)
			} else {
				log.Printf("code 0x%x shortPress", repeatedCode)
			}

			btn := codeToButton(repeatedCode, isLongPress)
			repeatedCode = 0
			repeatCnt = 0

			select {
			case buttonC <- btn:
			case <-ctx.Done():
				return
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
	for btn := range buttonC {
		log.Printf("button: %s", btn)
		go func(label string) {
			fnd.SetString(label)
			time.Sleep(1 * time.Second)
			fnd.SetString(fndBlank)
		}(btn.String())

		if err := tv.Handle(ctx, btn); err != nil {
			log.Printf("error doing %s: %v", btn, err)
		}
	}
}

func scanIRCodes(
	ctx context.Context,
	ser *serial.Port,
	codeInC chan<- uint32,
) {
	var lastCode uint32

	for ctx.Err() == nil {
		scanner := bufio.NewScanner(ser)
		for scanner.Scan() {
			codeStr := strings.TrimSpace(scanner.Text())
			if codeStr == "" {
				continue
			}

			code, isRepeat, err := parseIRCode(codeStr, lastCode)
			if err != nil {
				log.Printf("error parsing %q: %v", codeStr, err)
				continue
			}
			if !isRepeat {
				lastCode = code
			}
			if code == 0 {
				continue
			}

			select {
			case codeInC <- code:
			case <-ctx.Done():
				return
			}
		}

		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			log.Printf("error scanning serial port: %v", err)
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
