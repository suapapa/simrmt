package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/suapapa/go_webostv"
)

func TestBuildMagicPacket(t *testing.T) {
	t.Run("valid colon MAC", func(t *testing.T) {
		pkt, err := buildMagicPacket("aa:bb:cc:dd:ee:ff")
		if err != nil {
			t.Fatalf("buildMagicPacket() unexpected error: %v", err)
		}
		if len(pkt) != 102 {
			t.Fatalf("packet length = %d, want 102", len(pkt))
		}

		wantHeader := bytes.Repeat([]byte{0xff}, 6)
		if !bytes.Equal(pkt[:6], wantHeader) {
			t.Fatalf("header = %x, want %x", pkt[:6], wantHeader)
		}

		wantMAC := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		for i := range 16 {
			offset := 6 + i*6
			if !bytes.Equal(pkt[offset:offset+6], wantMAC) {
				t.Fatalf("repetition %d = %x, want %x", i, pkt[offset:offset+6], wantMAC)
			}
		}
	})

	t.Run("valid dash uppercase MAC", func(t *testing.T) {
		pkt, err := buildMagicPacket("11-22-33-44-55-66")
		if err != nil {
			t.Fatalf("buildMagicPacket() unexpected error: %v", err)
		}
		if len(pkt) != 102 {
			t.Fatalf("packet length = %d, want 102", len(pkt))
		}
	})

	invalidMACs := []string{
		"",
		"not-a-mac",
		"aa:bb:cc:dd:ee",
		"aa:bb:cc:dd:ee:ff:00:11",
	}
	for _, mac := range invalidMACs {
		t.Run("invalid_"+mac, func(t *testing.T) {
			if _, err := buildMagicPacket(mac); err == nil {
				t.Fatalf("buildMagicPacket(%q) expected error, got nil", mac)
			}
		})
	}
}

func TestParseARPTable(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		table   string
		wantMAC string
		wantErr bool
	}{
		{
			name: "linux proc net arp",
			ip:   "192.168.1.50",
			table: `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1      0x1         0x2         10:20:30:40:50:60     *        eth0
192.168.1.50     0x1         0x2         aa:bb:cc:dd:ee:ff     *        eth0
`,
			wantMAC: "aa:bb:cc:dd:ee:ff",
		},
		{
			name: "linux proc net arp ignores zero mac",
			ip:   "192.168.1.50",
			table: `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.50     0x1         0x0         00:00:00:00:00:00     *        eth0
`,
			wantErr: true,
		},
		{
			name: "linux arp -n",
			ip:   "192.168.1.50",
			table: `Address                  HWtype  HWaddress           Flags Mask            Iface
192.168.1.50             ether   AA:BB:CC:DD:EE:FF   C                     wlan0
`,
			wantMAC: "aa:bb:cc:dd:ee:ff",
		},
		{
			name:    "macos arp -n with single digit octets",
			ip:      "192.168.1.50",
			table:   "? (192.168.1.50) at a:bb:cc:d:ee:ff on en0 ifscope [ethernet]\n",
			wantMAC: "0a:bb:cc:0d:ee:ff",
		},
		{
			name:    "macos arp incomplete",
			ip:      "192.168.1.50",
			table:   "? (192.168.1.50) at (incomplete) on en0 ifscope [ethernet]\n",
			wantErr: true,
		},
		{
			name:    "ip not present",
			ip:      "192.168.1.99",
			table:   "? (192.168.1.50) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]\n",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseARPTable(tc.ip, tc.table)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseARPTable() err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.wantMAC {
				t.Fatalf("parseARPTable() = %q, want %q", got, tc.wantMAC)
			}
		})
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	// Missing store returns empty map without error.
	empty, err := loadStore(path)
	if err != nil {
		t.Fatalf("loadStore(missing) unexpected error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("loadStore(missing) len = %d, want 0", len(empty))
	}

	want := map[string]string{
		"host":       "192.168.219.181",
		"mac":        "aa:bb:cc:dd:ee:ff",
		"client_key": "5c930e4b77788374d70f42ab7d6f5685",
	}
	if err := saveStore(path, want); err != nil {
		t.Fatalf("saveStore() unexpected error: %v", err)
	}

	got, err := loadStore(path)
	if err != nil {
		t.Fatalf("loadStore(saved) unexpected error: %v", err)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("store[%q] = %q, want %q", k, got[k], v)
		}
	}

	// Corrupt JSON returns error.
	badPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("write corrupt store: %v", err)
	}
	if _, err := loadStore(badPath); err == nil {
		t.Fatal("loadStore(corrupt) expected error, got nil")
	}
}

func TestTVHandleDryRun(t *testing.T) {
	ctx := context.Background()
	storePath := filepath.Join(t.TempDir(), "store.json")
	tv := NewTV("", storePath, true)

	// AOUT cycles CurAudioOutIdx independently of CurAppIdx.
	if err := tv.Handle(ctx, AOUT); err != nil {
		t.Fatalf("Handle(AOUT #1) err = %v", err)
	}
	if tv.CurAudioOutIdx != 1 || tv.CurAppIdx != 0 {
		t.Fatalf("after AOUT #1: CurAudioOutIdx=%d CurAppIdx=%d, want 1, 0", tv.CurAudioOutIdx, tv.CurAppIdx)
	}

	if err := tv.Handle(ctx, AOUT); err != nil {
		t.Fatalf("Handle(AOUT #2) err = %v", err)
	}
	if tv.CurAudioOutIdx != 0 || tv.CurAppIdx != 0 {
		t.Fatalf("after AOUT #2: CurAudioOutIdx=%d CurAppIdx=%d, want 0, 0", tv.CurAudioOutIdx, tv.CurAppIdx)
	}

	// INPUT cycles CurAppIdx independently of CurAudioOutIdx.
	for i := 1; i <= len(tv.Apps); i++ {
		wantIdx := i % len(tv.Apps)
		if err := tv.Handle(ctx, INPUT); err != nil {
			t.Fatalf("Handle(INPUT #%d) err = %v", i, err)
		}
		if tv.CurAppIdx != wantIdx || tv.CurAudioOutIdx != 0 {
			t.Fatalf("after INPUT #%d: CurAppIdx=%d CurAudioOutIdx=%d, want %d, 0", i, tv.CurAppIdx, tv.CurAudioOutIdx, wantIdx)
		}
	}

	// Other buttons succeed and PWROFF updates Status.
	for _, btn := range []button{PWRON, UP, DOWN, LEFT, RIGHT, OK, BACK, HOME, CHUP, CHDOWN, VOLDOWN, VOLUP, INFO, UNKNOWN} {
		if err := tv.Handle(ctx, btn); err != nil {
			t.Fatalf("Handle(%s) err = %v", btn, err)
		}
	}
	if err := tv.Handle(ctx, PWROFF); err != nil {
		t.Fatalf("Handle(PWROFF) err = %v", err)
	}
	if tv.Status != tvStatusOff {
		t.Fatalf("Status = %v, want tvStatusOff", tv.Status)
	}
}

func TestTVLiveReconnectsAfterPointerOrMainClose(t *testing.T) {
	var (
		mu             sync.Mutex
		inputCommands  []string
		audioOutputs   []string
		pointerConns   []*websocket.Conn
		mainConns      []*websocket.Conn
		pointerSrvHost string
	)

	pointerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		mu.Lock()
		pointerConns = append(pointerConns, conn)
		mu.Unlock()
		defer func() { _ = conn.Close() }()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			mu.Lock()
			inputCommands = append(inputCommands, string(msg))
			mu.Unlock()
		}
	}))
	defer pointerSrv.Close()
	pointerSrvHost = "ws://" + strings.TrimPrefix(pointerSrv.URL, "http://") + "/pointer"

	mainSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		mu.Lock()
		mainConns = append(mainConns, conn)
		mu.Unlock()
		defer func() { _ = conn.Close() }()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg webostv.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}

			switch msg.Type {
			case "register":
				_ = conn.WriteJSON(webostv.Message{
					Type:    "registered",
					ID:      msg.ID,
					Payload: json.RawMessage(`{"client-key":"mock-key"}`),
				})
			case "request":
				switch msg.URI {
				case "ssap://com.webos.service.networkinput/getPointerInputSocket":
					_ = conn.WriteJSON(webostv.Message{
						Type:    "response",
						ID:      msg.ID,
						Payload: json.RawMessage(`{"socketPath":"` + pointerSrvHost + `"}`),
					})
				case "ssap://audio/changeSoundOutput":
					var p struct {
						Output string `json:"output"`
					}
					_ = json.Unmarshal(msg.Payload, &p)
					mu.Lock()
					audioOutputs = append(audioOutputs, p.Output)
					mu.Unlock()
					_ = conn.WriteJSON(webostv.Message{
						Type:    "response",
						ID:      msg.ID,
						Payload: json.RawMessage(`{"returnValue":true}`),
					})
				default:
					_ = conn.WriteJSON(webostv.Message{
						Type:    "response",
						ID:      msg.ID,
						Payload: json.RawMessage(`{"returnValue":true}`),
					})
				}
			}
		}
	}))
	defer mainSrv.Close()

	storePath := filepath.Join(t.TempDir(), "store.json")
	if err := saveStore(storePath, map[string]string{"client_key": "mock-key"}); err != nil {
		t.Fatalf("saveStore() err = %v", err)
	}

	tv := NewTV(strings.TrimPrefix(mainSrv.URL, "http://"), storePath, false)
	defer tv.Close()

	ctx := context.Background()
	if err := tv.Handle(ctx, UP); err != nil {
		t.Fatalf("Handle(UP) err = %v", err)
	}
	if err := tv.Handle(ctx, AOUT); err != nil {
		t.Fatalf("Handle(AOUT) err = %v", err)
	}

	// Simulate TV closing only the idle pointer socket.
	mu.Lock()
	pConn := pointerConns[0]
	mu.Unlock()
	_ = pConn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "idle"),
		time.Now().Add(time.Second),
	)
	_ = pConn.Close()
	time.Sleep(50 * time.Millisecond)

	if err := tv.Handle(ctx, RIGHT); err != nil {
		t.Fatalf("Handle(RIGHT) after pointer close err = %v", err)
	}

	// Simulate TV closing the main WebSocket connection too.
	mu.Lock()
	mConn := mainConns[0]
	mu.Unlock()
	_ = mConn.Close()
	time.Sleep(50 * time.Millisecond)

	if err := tv.Handle(ctx, LEFT); err != nil {
		t.Fatalf("Handle(LEFT) after main close err = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(inputCommands)
		mu.Unlock()
		if n >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(inputCommands) != 3 {
		t.Fatalf("got %d input commands, want 3: %v", len(inputCommands), inputCommands)
	}
	if !strings.Contains(inputCommands[0], "name:UP") ||
		!strings.Contains(inputCommands[1], "name:RIGHT") ||
		!strings.Contains(inputCommands[2], "name:LEFT") {
		t.Fatalf("unexpected input commands: %v", inputCommands)
	}
	if len(audioOutputs) != 1 || audioOutputs[0] != "tv_speaker" {
		t.Fatalf("audioOutputs = %v, want [tv_speaker]", audioOutputs)
	}
}
