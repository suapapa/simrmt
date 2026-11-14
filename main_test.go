package main

import (
	"context"
	"io"
	"testing"
	"time"
)

func TestParseIRCode(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		lastCode   uint32
		wantCode   uint32
		wantRepeat bool
		wantErr    bool
	}{
		{
			name:       "valid uppercase hex",
			input:      "FB04E786",
			lastCode:   0,
			wantCode:   0xFB04E786,
			wantRepeat: false,
		},
		{
			name:       "valid lowercase hex with whitespace",
			input:      "  bc43e786 \r",
			lastCode:   0,
			wantCode:   0xBC43E786,
			wantRepeat: false,
		},
		{
			name:       "repeat code uses lastCode",
			input:      "0",
			lastCode:   0xE817E786,
			wantCode:   0xE817E786,
			wantRepeat: true,
		},
		{
			name:       "repeat code with zero lastCode",
			input:      "0",
			lastCode:   0,
			wantCode:   0,
			wantRepeat: true,
		},
		{
			name:     "invalid hex characters",
			input:    "ZZZZZZZZ",
			lastCode: 0xFB04E786,
			wantErr:  true,
		},
		{
			name:     "too short hex code",
			input:    "FB04",
			lastCode: 0,
			wantErr:  true,
		},
		{
			name:     "too long hex code",
			input:    "FB04E78600",
			lastCode: 0,
			wantErr:  true,
		},
		{
			name:     "empty input",
			input:    "",
			lastCode: 0,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotCode, gotRepeat, err := parseIRCode(tc.input, tc.lastCode)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseIRCode(%q) err = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if gotCode != tc.wantCode {
				t.Fatalf("parseIRCode(%q) code = 0x%x, want 0x%x", tc.input, gotCode, tc.wantCode)
			}
			if gotRepeat != tc.wantRepeat {
				t.Fatalf("parseIRCode(%q) repeat = %v, want %v", tc.input, gotRepeat, tc.wantRepeat)
			}
		})
	}
}

func TestCodeToButton(t *testing.T) {
	tests := []struct {
		name      string
		code      uint32
		longPress bool
		want      button
	}{
		{name: "short PWRON", code: 0xFB04E786, longPress: false, want: PWRON},
		{name: "short BACK", code: 0xFF00E786, longPress: false, want: BACK},
		{name: "short AOUT", code: 0xBC43E786, longPress: false, want: AOUT},
		{name: "short UP", code: 0xE817E786, longPress: false, want: UP},
		{name: "short DOWN", code: 0xE41BE786, longPress: false, want: DOWN},
		{name: "short LEFT", code: 0xE51AE786, longPress: false, want: LEFT},
		{name: "short RIGHT", code: 0xA659E786, longPress: false, want: RIGHT},
		{name: "short OK", code: 0xBD42E786, longPress: false, want: OK},
		{name: "short INFO", code: 0xE916E786, longPress: false, want: INFO},
		{name: "short HOME", code: 0xF00FE786, longPress: false, want: HOME},

		{name: "long PWROFF", code: 0xFB04E786, longPress: true, want: PWROFF},
		{name: "long BACK", code: 0xFF00E786, longPress: true, want: BACK},
		{name: "long AOUT", code: 0xBC43E786, longPress: true, want: AOUT},
		{name: "long CHUP", code: 0xE817E786, longPress: true, want: CHUP},
		{name: "long CHDOWN", code: 0xE41BE786, longPress: true, want: CHDOWN},
		{name: "long VOLDOWN", code: 0xE51AE786, longPress: true, want: VOLDOWN},
		{name: "long VOLUP", code: 0xA659E786, longPress: true, want: VOLUP},
		{name: "long OK", code: 0xBD42E786, longPress: true, want: OK},
		{name: "long INFO", code: 0xE916E786, longPress: true, want: INFO},
		{name: "long HOME", code: 0xF00FE786, longPress: true, want: HOME},

		{name: "unknown short", code: 0x12345678, longPress: false, want: UNKNOWN},
		{name: "unknown long", code: 0x12345678, longPress: true, want: UNKNOWN},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codeToButton(tc.code, tc.longPress)
			if got != tc.want {
				t.Fatalf("codeToButton(0x%x, %v) = %v, want %v", tc.code, tc.longPress, got, tc.want)
			}
		})
	}
}

func TestPressState(t *testing.T) {
	const (
		upCode   uint32 = 0xE817E786
		downCode uint32 = 0xE41BE786
	)
	t0 := time.Unix(1000, 0)

	t.Run("single short tap", func(t *testing.T) {
		var ps pressState
		if got := ps.onEvent(irEvent{code: upCode, repeat: false}, t0); len(got) != 0 {
			t.Fatalf("unexpected immediate buttons: %v", got)
		}
		btn, ok := ps.onReleaseTimeout()
		if !ok || btn != UP {
			t.Fatalf("onReleaseTimeout() = (%v, %v), want (UP, true)", btn, ok)
		}
	})

	t.Run("short tap with one repeat", func(t *testing.T) {
		var ps pressState
		ps.onEvent(irEvent{code: upCode, repeat: false}, t0)
		if got := ps.onEvent(irEvent{code: upCode, repeat: true}, t0.Add(110*time.Millisecond)); len(got) != 0 {
			t.Fatalf("unexpected buttons on 1st repeat: %v", got)
		}
		btn, ok := ps.onReleaseTimeout()
		if !ok || btn != UP {
			t.Fatalf("onReleaseTimeout() = (%v, %v), want (UP, true)", btn, ok)
		}
	})

	t.Run("rapid taps of same key emit multiple short presses", func(t *testing.T) {
		var ps pressState
		var got []button

		for i := range 3 {
			now := t0.Add(time.Duration(i*150) * time.Millisecond)
			got = append(got, ps.onEvent(irEvent{code: upCode, repeat: false}, now)...)
		}
		if btn, ok := ps.onReleaseTimeout(); ok {
			got = append(got, btn)
		}

		want := []button{UP, UP, UP}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("got[%d] = %v, want %v (all=%v)", i, got[i], want[i], got)
			}
		}
	})

	t.Run("rapid alternating keys do not drop previous key", func(t *testing.T) {
		var ps pressState
		var got []button

		got = append(got, ps.onEvent(irEvent{code: upCode, repeat: false}, t0)...)
		got = append(got, ps.onEvent(irEvent{code: downCode, repeat: false}, t0.Add(50*time.Millisecond))...)
		if btn, ok := ps.onReleaseTimeout(); ok {
			got = append(got, btn)
		}

		want := []button{UP, DOWN}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("hold emits single long press immediately without extra short on release", func(t *testing.T) {
		var ps pressState
		var got []button

		got = append(got, ps.onEvent(irEvent{code: upCode, repeat: false}, t0)...)
		for i := 1; i <= 5; i++ {
			now := t0.Add(time.Duration(i*108) * time.Millisecond)
			got = append(got, ps.onEvent(irEvent{code: upCode, repeat: true}, now)...)
		}
		if btn, ok := ps.onReleaseTimeout(); ok {
			got = append(got, btn)
		}

		if len(got) != 1 || got[0] != CHUP {
			t.Fatalf("got %v, want [CHUP]", got)
		}
	})

	t.Run("stray repeat when idle is ignored", func(t *testing.T) {
		var ps pressState
		if got := ps.onEvent(irEvent{code: upCode, repeat: true}, t0); len(got) != 0 {
			t.Fatalf("expected stray repeat to be ignored, got %v", got)
		}
		if btn, ok := ps.onReleaseTimeout(); ok {
			t.Fatalf("expected no timeout button, got %v", btn)
		}
	})
}

func TestDetectPresses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	codeC := make(chan irEvent, 8)
	buttonC := make(chan button, 8)
	go detectPresses(ctx, codeC, buttonC)

	codeC <- irEvent{code: 0xE817E786, repeat: false}
	codeC <- irEvent{code: 0xE817E786, repeat: false}

	select {
	case btn := <-buttonC:
		if btn != UP {
			t.Fatalf("first button = %v, want UP", btn)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for immediate flushed UP")
	}

	select {
	case btn := <-buttonC:
		if btn != UP {
			t.Fatalf("second button = %v, want UP", btn)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second UP after release timeout")
	}
}

type chunkedEOFReader struct {
	chunks []string
	idx    int
}

func (r *chunkedEOFReader) Read(p []byte) (int, error) {
	if r.idx >= len(r.chunks) {
		return 0, io.EOF
	}
	chunk := r.chunks[r.idx]
	r.idx++
	if chunk == "" {
		return 0, io.EOF
	}
	n := copy(p, chunk)
	return n, nil
}

func TestScanIRCodesPreservesPartialLineAcrossEOF(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &chunkedEOFReader{
		chunks: []string{
			"",
			"E817",
			"",
			"E786\r\n",
			"0\r\n",
			"",
		},
	}
	codeInC := make(chan irEvent, 4)

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanIRCodes(ctx, reader, codeInC)
	}()

	var got []irEvent
	deadline := time.After(time.Second)
	for len(got) < 2 {
		select {
		case ev := <-codeInC:
			got = append(got, ev)
		case <-deadline:
			t.Fatalf("timed out waiting for parsed events, got %v", got)
		}
	}
	cancel()
	<-done

	if got[0] != (irEvent{code: 0xE817E786, repeat: false}) {
		t.Fatalf("got[0] = %+v, want {0xe817e786 false}", got[0])
	}
	if got[1] != (irEvent{code: 0xE817E786, repeat: true}) {
		t.Fatalf("got[1] = %+v, want {0xe817e786 true}", got[1])
	}
}
