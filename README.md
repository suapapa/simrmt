# simrmt

모니터에 딸려있던 막리모컨과 아두이노 IR 수신기, 라즈베리파이(TM1638 8자리 FND 포함)를 이용해 **LG WebOS TV의 사운드 출력(`tv_speaker` ↔ `external_optical`) 변환 등을 버튼 한 번으로 제어**하는 프로젝트입니다.

기존 [`simremote-ir`](https://github.com/suapapa/simremote-ir)을 [`go_webostv`](https://github.com/suapapa/go_webostv) 패키지를 사용해 순수 Go 단일 바이너리로 재구현했습니다.

## 배경 및 재구현 이유

LG OLED WebOS TV에서 광출력(5.1ch 리시버)과 TV 내장 스피커를 오가려면 `설정 -> 사운드` 메뉴를 깊이 들어가 좌우 버튼으로 고른 뒤 다시 Back 키로 닫아야 하는 번거로움이 있습니다.

이 문제를 해결하기 위해 막리모컨 + 아두이노 IR 수신기 + 라즈베리파이로 원버튼 리모컨을 만들었던 과정은 블로그 글에 정리되어 있습니다:

- 블로그 소개 글: [LG TV 사운드 출력 변환 한 버튼으로 하기](https://homin.dev/blog/post/20230228_simremote4lg-webos-tv/)

예전 구현 당시에는 Go용 WebOS 패키지의 완성도가 미묘하게 아쉬워서 Python의 `PyWebOSTV`로 HTTP API 서버를 띄우고 Go 프로그램에서 이를 호출하는 구조였습니다:

```text
기존: Arduino --serial--> Golang(simremote-ir) --http--> Python(simremote / PyWebOSTV) --ws--> LGTV
현재: Arduino --serial--> Golang(simrmt / go_webostv) --ws--> LGTV
```

이번 재구현에서는 `PyWebOSTV`를 Go로 직접 포팅한 [`github.com/suapapa/go_webostv`](https://github.com/suapapa/go_webostv)를 사용하므로, 별도의 Python HTTP 서버 없이 Go 프로그램 하나로 SSDP TV 검색, 페어링 키(`store.json`) 저장, Wake-on-LAN 전원 켜기, 키 입력, 사운드 출력 전환, 입력/앱 전환까지 직접 WebSocket으로 제어합니다.

### 기존 저장소

- [`suapapa/simremote-ir`](https://github.com/suapapa/simremote-ir): 기존 IR 수신 및 HTTP API 호출용 Go 프로그램
- [`suapapa/simremote`](https://github.com/suapapa/simremote): 기존 `PyWebOSTV` 기반 Python Flask HTTP 서버
- [`suapapa/go_webostv`](https://github.com/suapapa/go_webostv): 이번 재구현에서 직접 사용하는 Go WebOS TV 제어 패키지

## 하드웨어 구성

1. **Arduino + IR 수신 모듈** (`_arduino/irremote2serial.ino`)
   - `IR_RECEIVE_PIN`: `2`
   - `Arduino-IRremote` 라이브러리로 수신한 32비트 IR 코드(및 반복 코드 `0`)를 `9600` baud USB 시리얼로 전송
2. **Raspberry Pi + TM1638 8-digit FND**
   - 부팅 시 로컬 IP 끝자리를 `srmt-XXX` 형태로 표시
   - 리모컨 버튼 입력 시 해당 버튼 이름을 1초간 표시
   - GPIO 핀 연결 (`periph.io` BCM 핀 번호):
     - `DATA`: GPIO `17`
     - `CLK`: GPIO `27`
     - `STB`: GPIO `22`

## 버튼 매핑 (Short / Long Press)

버튼을 누르고 있는 동안 반복 시그널이 3회 이상 수신되고 첫 입력 후 400ms가 지나면 Long Press로 즉시 판정하며, 그 전에 손을 떼거나 다음 버튼이 입력되면 Short Press로 판정합니다 (`mode.go`).

| IR 코드 | Short Press | Long Press | 동작 |
| :--- | :--- | :--- | :--- |
| `0xFB04E786` | `PWRON` | `PWROFF` | Wake-on-LAN + 연결 / TV 전원 끄기 |
| `0xBC43E786` | `AOUT` | `AOUT` | 사운드 출력 순환 (`tv_speaker` ↔ `external_optical`) |
| `0xE916E786` | `INFO` | `INFO` | 정보 |
| `0xE817E786` | `UP` | `CHUP` | 위 / 채널 올림 |
| `0xE41BE786` | `DOWN` | `CHDOWN` | 아래 / 채널 내림 |
| `0xE51AE786` | `LEFT` | `VOLDOWN` | 왼쪽 / 볼륨 내림 |
| `0xA659E786` | `RIGHT` | `VOLUP` | 오른쪽 / 볼륨 올림 |
| `0xBD42E786` | `OK` | `OK` | 확인 |
| `0xFF00E786` | `BACK` | `BACK` | 뒤로 |
| `0xF00FE786` | `HOME` | `HOME` | 홈 |

## 빌드 및 실행

### 테스트 및 검증

```sh
go test -v -race ./...
```

### 라즈베리파이용 크로스 컴파일

```sh
GOOS=linux GOARCH=arm go build -o simrmt .
```

### 실행 옵션

```sh
./simrmt -port /dev/ttyUSB0 -store store.json
```

- `-port`: 아두이노가 연결된 시리얼 포트 경로 (기본값: `/dev/ttyUSB0`)
- `-tv`: TV IP 또는 호스트명 직접 지정 (비워 두면 SSDP로 자동 검색해 `store.json`에 캐시)
- `-store`: TV 호스트, MAC 주소(Wake-on-LAN용), 페어링 `client_key`를 저장할 JSON 파일 경로 (기본값: `store.json`)
- `-n`: Dry-run 모드 (GPIO 및 WebOS TV 네트워크 호출 없이 로그만 출력)

최초 실행 시 TV 화면에 페어링 승인 팝업이 뜨면 수락해 주어야 하며, 이후 발급된 `client_key`가 `store.json`에 저장되어 재연결 시에는 팝업 없이 자동으로 연결됩니다.
