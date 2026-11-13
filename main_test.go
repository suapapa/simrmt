package main

import "testing"

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
		{name: "short INPUT", code: 0xE916E786, longPress: false, want: INPUT},
		{name: "short HOME", code: 0xF00FE786, longPress: false, want: HOME},

		{name: "long PWROFF", code: 0xFB04E786, longPress: true, want: PWROFF},
		{name: "long BACK", code: 0xFF00E786, longPress: true, want: BACK},
		{name: "long AOUT", code: 0xBC43E786, longPress: true, want: AOUT},
		{name: "long CHUP", code: 0xE817E786, longPress: true, want: CHUP},
		{name: "long CHDOWN", code: 0xE41BE786, longPress: true, want: CHDOWN},
		{name: "long VOLDOWN", code: 0xE51AE786, longPress: true, want: VOLDOWN},
		{name: "long VOLUP", code: 0xA659E786, longPress: true, want: VOLUP},
		{name: "long OK", code: 0xBD42E786, longPress: true, want: OK},
		{name: "long INPUT", code: 0xE916E786, longPress: true, want: INPUT},
		{name: "long INFO", code: 0xF00FE786, longPress: true, want: INFO},

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
