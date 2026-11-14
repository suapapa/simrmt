package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/suapapa/go_webostv"
)

const (
	commandTimeout = 10 * time.Second
	pairingTimeout = 30 * time.Second
)

type tvStatus uint8

const (
	tvStatusUnknown tvStatus = iota
	tvStatusOn
	tvStatusOff
)

// TV manages the WebOS TV connection, pairing store, and remote control actions.
type TV struct {
	Status         tvStatus
	AudioOuts      []string
	CurAudioOutIdx int
	Apps           []string
	CurAppIdx      int

	tvHost    string
	storePath string
	store     map[string]string
	dryRun    bool

	mu        sync.Mutex
	connected bool
	client    *webostv.Client
	mediaCtl  *webostv.MediaControl
	tvCtl     *webostv.TvControl
	systemCtl *webostv.SystemControl
	appCtl    *webostv.ApplicationControl
	inputCtl  *webostv.InputControl
	sourceCtl *webostv.SourceControl
}

// NewTV creates a TV controller backed by go_webostv and loads any persisted pairing store.
func NewTV(tvHost, storePath string, dryRun bool) *TV {
	store, err := loadStore(storePath)
	if err != nil {
		log.Printf("warning: failed to load store %q: %v", storePath, err)
		store = map[string]string{}
	}

	return &TV{
		AudioOuts: []string{
			"tv_speaker",
			"external_optical",
		},
		Apps: []string{
			"현재 방송",
			"HDMI1",
			"HDMI2",
		},
		tvHost:    tvHost,
		storePath: storePath,
		store:     store,
		dryRun:    dryRun,
	}
}

// Handle executes the WebOS TV action corresponding to the given remote button.
func (t *TV) Handle(ctx context.Context, btn button) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.dryRun {
		return t.handleDryRun(btn)
	}

	timeout := commandTimeout
	if !t.connected && t.store["client_key"] == "" {
		timeout = pairingTimeout
	}
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch btn {
	case UP, DOWN, LEFT, RIGHT, OK, BACK, HOME, CHUP, CHDOWN, VOLDOWN, VOLUP, INFO:
		return t.keyHandler(opCtx, btn)
	case PWRON:
		return t.powerOn(opCtx)
	case PWROFF:
		return t.powerOff(opCtx)
	case AOUT:
		return t.cycleAudioOut(opCtx)
	case INPUT:
		return t.cycleApp(opCtx)
	default:
		return nil
	}
}

// Close disconnects the WebOS client and pointer input socket.
func (t *TV) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.disconnect()
}

func (t *TV) handleDryRun(btn button) error {
	switch btn {
	case UP, DOWN, LEFT, RIGHT, OK, BACK, HOME, CHUP, CHDOWN, VOLDOWN, VOLUP, INFO:
		log.Printf("dry run: send key %s", btn)
	case PWRON:
		log.Println("dry run: power on TV")
	case PWROFF:
		log.Println("dry run: power off TV")
		t.Status = tvStatusOff
	case AOUT:
		out := t.AudioOuts[t.CurAudioOutIdx]
		log.Printf("dry run: set audio output to %s", out)
		t.CurAudioOutIdx = (t.CurAudioOutIdx + 1) % len(t.AudioOuts)
	case INPUT:
		appTarget := t.Apps[t.CurAppIdx]
		log.Printf("dry run: launch app %s", appTarget)
		t.CurAppIdx = (t.CurAppIdx + 1) % len(t.Apps)
	default:
		log.Printf("dry run: ignore button %s", btn)
	}
	return nil
}

func (t *TV) connect(ctx context.Context) error {
	if t.connected {
		return nil
	}

	if err := t.ensureHost(ctx); err != nil {
		return err
	}

	host := t.store["host"]
	log.Printf("connecting to TV at %s...", host)

	client, err := t.dialTV(host)
	if err != nil {
		return fmt.Errorf("connect TV %s: %w", host, err)
	}

	if t.store["mac"] == "" {
		if mac, err := resolveMAC(host); err == nil {
			t.store["mac"] = mac
			if saveErr := saveStore(t.storePath, t.store); saveErr != nil {
				log.Printf("warning: failed to save MAC to store: %v", saveErr)
			}
		} else {
			log.Printf("warning: could not resolve MAC for %s: %v", host, err)
		}
	}

	if err := t.register(ctx, client); err != nil {
		_ = client.Close()
		return err
	}

	inputCtl := &webostv.InputControl{Control: webostv.Control{Client: client}}
	if err := inputCtl.ConnectInput(ctx); err != nil {
		_ = client.Close()
		return fmt.Errorf("connect input socket: %w", err)
	}

	t.client = client
	t.mediaCtl = &webostv.MediaControl{Control: webostv.Control{Client: client}}
	t.tvCtl = &webostv.TvControl{Control: webostv.Control{Client: client}}
	t.systemCtl = &webostv.SystemControl{Control: webostv.Control{Client: client}}
	t.appCtl = &webostv.ApplicationControl{Control: webostv.Control{Client: client}}
	t.inputCtl = inputCtl
	t.sourceCtl = &webostv.SourceControl{Control: webostv.Control{Client: client}}
	t.connected = true
	t.Status = tvStatusOn
	return nil
}

func (t *TV) dialTV(host string) (*webostv.Client, error) {
	if t.store["secure"] == "true" {
		client := webostv.NewClient(host, webostv.WithSecure(true))
		if err := client.Connect(); err != nil {
			return nil, err
		}
		return client, nil
	}

	client := webostv.NewClient(host)
	err := client.Connect()
	if err == nil {
		return client, nil
	}

	log.Printf("insecure connect to %s failed (%v), retrying with TLS...", host, err)
	secureClient := webostv.NewClient(host, webostv.WithSecure(true))
	if secureErr := secureClient.Connect(); secureErr != nil {
		return nil, err
	}
	t.store["secure"] = "true"
	if saveErr := saveStore(t.storePath, t.store); saveErr != nil {
		log.Printf("warning: failed to save secure flag to store: %v", saveErr)
	}
	return secureClient, nil
}

func (t *TV) ensureHost(ctx context.Context) error {
	if t.tvHost != "" {
		t.store["host"] = t.tvHost
		return nil
	}
	if t.store["host"] != "" {
		return nil
	}

	log.Println("discovering WebOS TV...")
	clients, err := webostv.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discover TV: %w", err)
	}
	if len(clients) == 0 {
		return errors.New("no WebOS TV found")
	}

	u, err := url.Parse(clients[0].URL())
	if err != nil {
		return fmt.Errorf("parse discovered TV URL %q: %w", clients[0].URL(), err)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname in discovered TV URL %q", clients[0].URL())
	}

	t.store["host"] = host
	if err := saveStore(t.storePath, t.store); err != nil {
		log.Printf("warning: failed to save discovered host: %v", err)
	}
	return nil
}

func (t *TV) register(ctx context.Context, client *webostv.Client) error {
	statusChan, errChan := client.Register(ctx, t.store)

	registered := false
	for statusChan != nil || errChan != nil {
		select {
		case status, ok := <-statusChan:
			if !ok {
				statusChan = nil
				continue
			}
			switch status {
			case webostv.Prompted:
				log.Println("Please accept the connection on the TV!")
			case webostv.Registered:
				log.Println("Registration successful!")
				registered = true
				if err := saveStore(t.storePath, t.store); err != nil {
					log.Printf("warning: failed to save client key: %v", err)
				}
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				continue
			}
			if err != nil {
				return fmt.Errorf("register TV: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if !registered {
		return errors.New("TV registration did not complete")
	}
	return nil
}

func (t *TV) disconnect() {
	if t.inputCtl != nil {
		_ = t.inputCtl.DisconnectInput()
	}
	if t.client != nil {
		_ = t.client.Close()
	}
	t.connected = false
	t.client = nil
	t.mediaCtl = nil
	t.tvCtl = nil
	t.systemCtl = nil
	t.appCtl = nil
	t.inputCtl = nil
	t.sourceCtl = nil
}

func (t *TV) keyHandler(ctx context.Context, btn button) error {
	if err := t.connect(ctx); err != nil {
		return err
	}

	var err error
	switch btn {
	case UP:
		err = t.inputCtl.Up()
	case DOWN:
		err = t.inputCtl.Down()
	case LEFT:
		err = t.inputCtl.Left()
	case RIGHT:
		err = t.inputCtl.Right()
	case OK:
		err = t.inputCtl.OK()
	case BACK:
		err = t.inputCtl.Back()
	case HOME:
		err = t.inputCtl.Home()
	case CHUP:
		err = t.inputCtl.ChannelUp()
	case CHDOWN:
		err = t.inputCtl.ChannelDown()
	case VOLDOWN:
		err = t.inputCtl.VolumeDown()
	case VOLUP:
		err = t.inputCtl.VolumeUp()
	case INFO:
		err = t.inputCtl.Info()
	default:
		return fmt.Errorf("unknown button: %s", btn)
	}

	if err != nil {
		t.disconnect()
		return fmt.Errorf("send key %s: %w", btn, err)
	}
	return nil
}

func (t *TV) powerOn(ctx context.Context) error {
	if mac := t.store["mac"]; mac != "" {
		log.Printf("sending Wake-on-LAN to %s...", mac)
		if err := sendWakeOnLAN(mac); err != nil {
			return fmt.Errorf("turn on TV: %w", err)
		}
	}
	if err := t.connect(ctx); err != nil {
		return fmt.Errorf("turn on TV: %w", err)
	}
	return nil
}

func (t *TV) powerOff(ctx context.Context) error {
	if err := t.connect(ctx); err != nil {
		if isTimeoutOrClosed(err) {
			t.Status = tvStatusOff
			return nil
		}
		return fmt.Errorf("turn off TV: %w", err)
	}

	err := t.systemCtl.PowerOff(ctx)
	t.disconnect()
	if err != nil {
		if isTimeoutOrClosed(err) {
			t.Status = tvStatusOff
			return nil
		}
		return fmt.Errorf("turn off TV: %w", err)
	}

	t.Status = tvStatusOff
	return nil
}

func (t *TV) cycleAudioOut(ctx context.Context) error {
	if err := t.connect(ctx); err != nil {
		return fmt.Errorf("change audio output: %w", err)
	}

	out := t.AudioOuts[t.CurAudioOutIdx]
	if err := t.mediaCtl.SetAudioOutput(ctx, out); err != nil {
		t.disconnect()
		return fmt.Errorf("set audio output %q: %w", out, err)
	}

	_ = t.systemCtl.Notify(
		ctx,
		fmt.Sprintf("Set audio output to %s", out),
		nil,
		"",
	)
	t.CurAudioOutIdx = (t.CurAudioOutIdx + 1) % len(t.AudioOuts)
	return nil
}

func (t *TV) cycleApp(ctx context.Context) error {
	if err := t.connect(ctx); err != nil {
		return fmt.Errorf("change app: %w", err)
	}

	apps, err := t.appCtl.ListApps(ctx)
	if err != nil {
		t.disconnect()
		return fmt.Errorf("list apps: %w", err)
	}

	target := t.Apps[t.CurAppIdx]
	matched, ok := findApp(apps, target)
	if !ok {
		return fmt.Errorf("application %q not found", target)
	}

	if err := t.appCtl.Launch(ctx, matched.ID, "", nil); err != nil {
		t.disconnect()
		return fmt.Errorf("launch app %q: %w", target, err)
	}

	_ = t.systemCtl.Notify(
		ctx,
		fmt.Sprintf("Set app to %s", target),
		nil,
		"",
	)
	t.CurAppIdx = (t.CurAppIdx + 1) % len(t.Apps)
	return nil
}

func findApp(apps []webostv.Application, target string) (webostv.Application, bool) {
	for _, app := range apps {
		if app.Title == target || app.ID == target {
			return app, true
		}
	}
	return webostv.Application{}, false
}

func isTimeoutOrClosed(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, webostv.ErrConnectionClosed) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func loadStore(path string) (map[string]string, error) {
	if path == "" {
		return map[string]string{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	store := map[string]string{}
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store, nil
}

func saveStore(path string, store map[string]string) error {
	if path == "" {
		return nil
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".store-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
