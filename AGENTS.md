# AGENTS.md

Guidelines for coding agents working in the `simrmt` repository.

## Project Overview

`simrmt` is a pure-Go reimplementation of [`simremote-ir`](https://github.com/suapapa/simremote-ir) using [`github.com/suapapa/go_webostv`](https://github.com/suapapa/go_webostv). It receives 32-bit hex IR remote codes over USB serial (`9600` baud) from an Arduino (`_arduino/irremote2serial.ino`), renders startup IP suffix (`srmt-XXX`) and button labels on a TM1638 8-digit 7-segment FND module connected to Raspberry Pi GPIO pins (data=`17`, clk=`27`, stb=`22`), and controls an LG WebOS TV directly over WebSocket.

This replaces the older two-process architecture (`simremote-ir` -> HTTP -> Python `simremote`/`PyWebOSTV` -> WebSocket) with a single Go binary.

## Repository Structure

- `main.go`, `main_test.go`: CLI entrypoint (`parseFlags` for `-port`, `-tv`, `-store`, `-n`), `SIGINT`/`SIGTERM` signal handling, serial IR scanner (`scanIRCodes`, `parseIRCode`), short/long press detector (`detectPresses`, `codeToButton`), and button consumer (`handleButtons`). Keep `main.go` at the repository root so `go build .` produces the binary.
- `button.go`, `mode.go`: `button` enum (`String()`) and `shortPressButton` / `longPressButton` IR code maps (`uint32` -> `button`).
- `tv.go`, `tv_test.go`: `TV` controller managing SSDP discovery (`webostv.Discover`), WebSocket connection and pairing (`Client.Register`), atomic JSON persistence (`loadStore`, `saveStore`), `go_webostv` controls (`MediaControl`, `SystemControl`, `ApplicationControl`, `InputControl`, `TvControl`, `SourceControl`), independent cycling of `AudioOuts` (`CurAudioOutIdx`) and `Apps` (`CurAppIdx`), and `-n` dry-run handling.
- `wol.go`: Wake-on-LAN UDP broadcast magic packet builder/sender (`buildMagicPacket`, `sendWakeOnLAN`) and ARP-based MAC lookup (`resolveMAC`, `parseARPTable` supporting `/proc/net/arp` and `arp -n`).
- `fnd.go`: `FND` wrapper around `github.com/suapapa/go_devices/tm1638` and `periph.io` GPIO, with `-n` dry-run log-only fallback.
- `_arduino/irremote2serial.ino`: Arduino sketch using `IRremote.hpp` on `IR_RECEIVE_PIN 2`, printing raw hex codes (and `0` on repeat) at `9600` baud.

## Build & Verification Commands

Run formatting, static analysis, and race-enabled tests before finishing any Go change:

```sh
gofmt -l .
go vet ./...
go test -race ./...
```

Cross-compile for Raspberry Pi (32-bit ARM):

```sh
GOOS=linux GOARCH=arm go build -o simrmt .
```

## Runtime Flags, Store & Hardware Boundaries

- **CLI flags**:
  - `-port` (default `/dev/ttyUSB0`): serial port connected to Arduino.
  - `-tv` (default `""`): optional TV IP/hostname override; if empty, SSDP discovery caches the host in the store.
  - `-store` (default `store.json`): JSON file storing `host`, `mac`, and `client_key`.
  - `-n` (default `false`): dry-run mode; skips GPIO initialization and WebOS TV network calls, logging actions instead.
- **Pairing store**: `store.json` is gitignored. First connection without `client_key` triggers `webostv.Prompted` and waits up to `pairingTimeout` (`30s`) for the user to accept the prompt on the TV. Normal commands use `commandTimeout` (`10s`).
- **Offline testability**: Serial ports, Raspberry Pi GPIO (`periph.io`), and a live LG WebOS TV are unavailable in standard dev/CI environments. Keep `go test ./...` hardware-free by testing pure logic (`parseIRCode`, `codeToButton`, `buildMagicPacket`, `parseARPTable`, `loadStore`/`saveStore`, `TV.Handle` dry-run) or using local mock servers.

## Known Future Improvement Areas

- Making `AudioOuts` and `Apps` configurable (e.g., YAML config or querying TV inputs/apps) instead of hardcoded defaults in `NewTV`.
- Adding a `systemd` service unit file for Raspberry Pi deployment.
- Adding WebSocket mock tests covering `TV.connect` and live `TV.Handle` command paths.

## Go Code Style

- Break lines beyond ~120 characters at semantic boundaries; place 4+ call arguments and multi-line signature parameters one per line.
- Use `:=` for non-zero values and `var` for zero-value initialization; initialize slices and maps explicitly (never nil); always use field names in struct composite literals.
- Handle errors and edge cases first with early returns; drop `else` after `return`/`break`/`continue`; extract 3+ operand conditions into named booleans.
- Keep functions focused (`<=4` parameters where practical) and prefer `range` loops.
