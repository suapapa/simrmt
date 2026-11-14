window.SIMRMT_I18N = (function () {
  var SUPPORTED = ["ko", "en", "ja", "zh"];
  var STORAGE_KEY = "simrmt-lang";

  var strings = {
    ko: {
      docTitle: "simrmt - LG WebOS TV 원버튼 IR 리모컨",
      metaDescription:
        "막리모컨과 아두이노, 라즈베리파이로 LG WebOS TV 사운드 출력과 입력을 버튼 한 번에 바꾸는 순수 Go 프로젝트입니다.",
      navAria: "주요 탐색",
      langAria: "언어 선택",
      navStory: "이야기",
      navArch: "구조",
      navHardware: "하드웨어",
      navDemo: "영상",
      navGithub: "GitHub",
      heroTitle: "사운드 출력 전환,<br>버튼 한 번으로 끝",
      heroLead:
        "LG WebOS TV의 내장 스피커와 광출력(5.1ch 리시버)을 오가려고 매번 설정 메뉴 깊숙이 들어가지 않아도 됩니다. 막리모컨, 아두이노, 라즈베리파이와 순수 Go 단일 바이너리로 만든 원버튼 리모컨입니다.",
      heroCtaGithub: "GitHub 보기",
      heroCtaArch: "구조 살펴보기",
      storyTitle: "설정 메뉴가 답답해서<br>직접 만들었습니다",
      storyP1:
        "LG OLED TV 광출력 단자에 15년 넘은 5.1ch 리시버를 물려두면 내장 OTT 앱 소리가 5.1ch로 깔끔하게 나옵니다. 문제는 TV 스피커와 광출력을 오갈 때마다 설정에서 사운드 메뉴로 깊이 들어가 좌우 키로 고르고 Back으로 닫아야 한다는 점이었습니다.",
      storyP2:
        "예전에 Go용 WebOS 패키지가 마땅치 않을 때는 PyWebOSTV로 Python HTTP 서버를 띄우고 Go 프로그램(simremote-ir)이 호출하도록 얽어 두었습니다. 부서진 모니터에서 얻은 막리모컨과 IR 수신기를 아두이노에 붙이고, 라즈베리파이에서 동작하게 만든 게 시작이었습니다.",
      storyP3:
        "이번에는 PyWebOSTV를 직접 Go로 옮긴 go_webostv를 써서 Python 서버를 걷어내고 Go 바이너리 하나로 합쳤습니다. webOS25의 TLS(wss://:3001) 연결과 Wake-on-LAN까지 한 번에 처리합니다.",
      storyCapSketch: "2023년 처음 떠올렸던 컨셉 스케치",
      archTitle: "Python 브리지를 걷어낸<br>단일 Go 구조",
      archLead:
        "두 프로세스로 나뉘어 HTTP를 타던 예전 구조를 단일 Go 바이너리와 직접 WebSocket/TLS 연결로 줄였습니다.",
      archLegacyTitle: "기존 구조 (simremote-ir + simremote)",
      archLegacyNote: "시리얼 파싱용 Go 프로세스와 PyWebOSTV용 Python Flask 서버 두 개를 함께 띄워야 했습니다.",
      archCurrentTitle: "현재 구조 (simrmt + go_webostv)",
      archCurrentNote: "아두이노 시리얼 수신부터 SSDP 검색, WoL, WebOS 제어, FND 출력까지 한 바이너리에서 끝납니다.",
      archPreviewAlt: "simrmt 아키텍처 다이어그램 미리보기",
      archInteractiveTitle: "전체 코드 기반 아키텍처 보기",
      archInteractiveBody:
        "main.go, tv.go, fnd.go, wol.go의 실제 호출 흐름과 소스 위치를 대화형 다이어그램으로 열어 볼 수 있습니다.",
      archInteractiveCta: "대화형 아키텍처 열기",
      hwTitle: "하드웨어 배선과<br>버튼 매핑",
      hwLead:
        "아두이노가 NEC IR 코드를 읽어 USB 시리얼로 넘기면, 라즈베리파이의 simrmt가 짧게·길게 누름을 판정해 TM1638 FND와 TV로 보냅니다.",
      pinTableTitle: "핀 연결",
      colDevice: "장치",
      colSignal: "역할 / 핀",
      colTarget: "연결 대상",
      pinArduino: "Arduino + IR 수신기",
      pinArduinoSig: "IR_RECEIVE_PIN 2 · 9600 baud",
      pinArduinoDst: "USB 시리얼 (/dev/ttyUSB0)",
      pinFndData: "TM1638 FND (DATA)",
      pinFndDataSig: "GPIO 17 (BCM)",
      pinFndClk: "TM1638 FND (CLK)",
      pinFndClkSig: "GPIO 27 (BCM)",
      pinFndStb: "TM1638 FND (STB)",
      pinFndStbSig: "GPIO 22 (BCM)",
      pinFndDst: "라즈베리파이 GPIO (periph.io)",
      btnTableTitle: "IR 코드별 동작",
      colHex: "IR 코드",
      colShort: "짧게 누르기",
      colLong: "길게 누르기",
      colEffect: "실제 동작",
      deployTitle: "크로스 빌드와 systemd 서비스 설치",
      deployBody:
        "라즈베리파이 1 B+(ARMv6)부터 64비트 보드까지 스크립트 한 줄로 크로스 컴파일하고 systemd 서비스로 등록합니다.",
      cliTitle: "실행 플래그",
      cliBody:
        "-port(기본 /dev/ttyUSB0), -tv(미지정 시 SSDP 자동 검색), -store(호스트·MAC·client_key·TLS 여부 저장), -n(드라이런)",
      demoTitle: "실제 TV에서 동작하는 모습",
      demoLead:
        "막리모컨 버튼 한 번으로 TV 내장 스피커와 5.1ch 광출력 리시버를 오가는 시연 영상과 관련 저장소입니다.",
      demoVideoAlt: "simrmt LG WebOS TV 동작 시연 영상 썸네일",
      demoWatchCta: "YouTube에서 영상 보기",
      demoCardTitle: "원버튼 사운드 전환 시연",
      demoCardBody:
        "원래 블로그 글에 올렸던 실제 거실 테스트 영상입니다. 2023년 개념 그대로 단일 Go 바이너리로 이어졌습니다.",
      linkBlog: "블로그 원문",
      linkBlogDesc: "LG TV 사운드 출력 변환 한 버튼으로 하기",
      linkRepoDesc: "단일 Go 바이너리 & systemd 배포 스크립트",
      linkWebosDesc: "PyWebOSTV를 포팅한 순수 Go WebOS 클라이언트",
      linkDevDesc: "TM1638 8자리 FND 제어 패키지",
      footerCopy: "Homin Lee <i@homin.dev> · MIT License",
      footerBackTop: "맨 위로"
    },
    en: {
      docTitle: "simrmt - One-Button IR Remote for LG WebOS TV",
      metaDescription:
        "Switch LG WebOS TV audio outputs and inputs with a single button using a spare IR remote, Arduino, Raspberry Pi, and a single Go binary.",
      navAria: "Primary navigation",
      langAria: "Language selector",
      navStory: "Story",
      navArch: "Architecture",
      navHardware: "Hardware",
      navDemo: "Demo",
      navGithub: "GitHub",
      heroTitle: "Switch TV audio,<br>with one button",
      heroLead:
        "Stop digging through LG WebOS menus just to toggle between TV speakers and your 5.1ch optical receiver. Built with a spare IR remote, an Arduino, a Raspberry Pi, and a single pure-Go binary.",
      heroCtaGithub: "View on GitHub",
      heroCtaArch: "See Architecture",
      storyTitle: "Built to escape<br>deep sound menus",
      storyP1:
        "Connecting an LG OLED TV to a 15-year-old 5.1ch receiver over optical S/PDIF gives clean surround sound in built-in OTT apps. The catch is switching back and forth: you had to open Settings, go into Sound, toggle left/right, and hit Back to dismiss the menu.",
      storyP2:
        "When existing Go WebOS libraries weren't quite complete, the first setup chained a Go serial reader (simremote-ir) over HTTP to a Python Flask bridge (simremote) running PyWebOSTV, driven by an IR receiver salvaged from a broken monitor.",
      storyP3:
        "Now PyWebOSTV has been ported directly to Go as go_webostv, collapsing both processes into a single Go binary (simrmt) with SSDP discovery, Wake-on-LAN, TM1638 FND feedback, and webOS25 wss://:3001 TLS support.",
      storyCapSketch: "Original 2023 concept drawing from the blog post",
      archTitle: "From two-process bridge<br>to a single Go binary",
      archLead:
        "Removed the local HTTP hop and Python runtime by speaking SSAP directly over WebSocket and TLS.",
      archLegacyTitle: "Before (simremote-ir + simremote)",
      archLegacyNote: "Required running a Go serial daemon and a Python Flask server side by side on the Raspberry Pi.",
      archCurrentTitle: "Now (simrmt + go_webostv)",
      archCurrentNote: "Serial scanning, press timing, FND output, WoL, and WebOS control live in one static binary.",
      archPreviewAlt: "Preview of the interactive simrmt architecture diagram",
      archInteractiveTitle: "Explore the code-backed architecture",
      archInteractiveBody:
        "Inspect every module from main.go, tv.go, fnd.go, and wol.go with traced call paths and source locations.",
      archInteractiveCta: "Open interactive diagram",
      hwTitle: "Hardware wiring &<br>button mapping",
      hwLead:
        "Arduino prints 32-bit IR codes over USB serial; simrmt separates short and long presses and drives both the TM1638 display and the TV.",
      pinTableTitle: "GPIO & Serial Wiring",
      colDevice: "Part",
      colSignal: "Pin / Config",
      colTarget: "Connection",
      pinArduino: "Arduino + IR Module",
      pinArduinoSig: "IR_RECEIVE_PIN 2 · 9600 baud",
      pinArduinoDst: "USB Serial (/dev/ttyUSB0)",
      pinFndData: "TM1638 FND (DATA)",
      pinFndDataSig: "GPIO 17 (BCM)",
      pinFndClk: "TM1638 FND (CLK)",
      pinFndClkSig: "GPIO 27 (BCM)",
      pinFndStb: "TM1638 FND (STB)",
      pinFndStbSig: "GPIO 22 (BCM)",
      pinFndDst: "Raspberry Pi GPIO (periph.io)",
      btnTableTitle: "Short & Long Press Map",
      colHex: "IR Code",
      colShort: "Short Press",
      colLong: "Long Press",
      colEffect: "Behavior",
      deployTitle: "Cross-compile & install as a systemd service",
      deployBody:
        "Scripts support both 32-bit ARMv6/v7 (Raspberry Pi 1 B+ through Pi 4) and 64-bit ARM boards.",
      cliTitle: "Runtime flags",
      cliBody:
        "-port (default /dev/ttyUSB0), -tv (optional IP override), -store (host, MAC, client_key, TLS cache), -n (dry-run).",
      demoTitle: "In action on a real TV",
      demoLead:
        "Watch the one-button audio output switch in the living room and check the companion Go repositories.",
      demoVideoAlt: "YouTube demo thumbnail showing simrmt controlling an LG WebOS TV",
      demoWatchCta: "Watch on YouTube",
      demoCardTitle: "Living room demo",
      demoCardBody:
        "Recorded with the original 2023 prototype; the exact same IR codes and workflow now run on the pure-Go binary.",
      linkBlog: "Original Blog Post",
      linkBlogDesc: "LG TV 사운드 출력 변환 한 버튼으로 하기 (Korean)",
      linkRepoDesc: "Single Go binary & systemd install scripts",
      linkWebosDesc: "Pure-Go WebOS TV client ported from PyWebOSTV",
      linkDevDesc: "TM1638 8-digit 7-segment display driver",
      footerCopy: "Homin Lee <i@homin.dev> · MIT License",
      footerBackTop: "Back to top"
    },
    ja: {
      docTitle: "simrmt - LG WebOS TV ワンボタンIRリモコン",
      metaDescription:
        "余ったIRリモコンとArduino、Raspberry Pi、単一のGoバイナリでLG WebOS TVの音声出力と入力をワンボタンで切り替えるプロジェクトです。",
      navAria: "メインナビゲーション",
      langAria: "言語切替",
      navStory: "背景",
      navArch: "構成",
      navHardware: "ハードウェア",
      navDemo: "動画",
      navGithub: "GitHub",
      heroTitle: "TV音声出力の切替を<br>ボタンひとつで",
      heroLead:
        "LG WebOS TVの内蔵スピーカーと光デジタル出力（5.1chレシーバー）を切り替えるたび、深い設定メニューを開く必要はありません。余ったIRリモコン、Arduino、Raspberry Pi、そして純粋なGo単一バイナリで実現しました。",
      heroCtaGithub: "GitHubを見る",
      heroCtaArch: "構成を見る",
      storyTitle: "設定メニューが面倒で<br>自分で作りました",
      storyP1:
        "LG OLED TVの光デジタル出力に15年前の5.1chレシーバーをつなぐと、内蔵OTTアプリの音声が5.1chで楽しめます。ただ、内蔵スピーカーと光出力を切り替えるたびに「設定 → サウンド」の奥まで移動し、左右キーで選んでBackで閉じる手間がありました。",
      storyP2:
        "以前はGo向けのWebOSライブラリが惜しかったため、GoのIR受信プログラム（simremote-ir）からHTTP経由でPython Flaskサーバー（simremote / PyWebOSTV）を叩く2プロセス構成で動かしていました。",
      storyP3:
        "今回PyWebOSTVをGoへ移植した go_webostv を作り、Pythonサーバーをなくして単一のGoバイナリ（simrmt）に統合しました。SSDP自動検出、Wake-on-LAN、webOS25向けのTLS（wss://:3001）接続まで1つで完結します。",
      storyCapSketch: "2023年公開当時の手描きコンセプト図",
      archTitle: "Pythonブリッジを廃した<br>単一Goバイナリ構成",
      archLead:
        "プロセス間のHTTP通信をなくし、GoからWebSocket / TLSで直接TVを制御します。",
      archLegacyTitle: "旧構成 (simremote-ir + simremote)",
      archLegacyNote: "シリアル読取のGoプロセスとPyWebOSTV用Python Flaskサーバーの2つを常駐させていました。",
      archCurrentTitle: "新構成 (simrmt + go_webostv)",
      archCurrentNote: "シリアル受信、長短押し判定、TM1638表示、WoL、SSAP制御を単一バイナリで実行します。",
      archPreviewAlt: "simrmt インタラクティブ構成図のプレビュー",
      archInteractiveTitle: "コード連動アーキテクチャを見る",
      archInteractiveBody:
        "main.go、tv.go、fnd.go、wol.go のデータフローと該当ソース行をインタラクティブ図で確認できます。",
      archInteractiveCta: "構成図を開く",
      hwTitle: "配線と<br>ボタン割り当て",
      hwLead:
        "Arduinoが32bit IRコードをUSBシリアルへ送り、simrmtが短押し・長押しを判別してFNDとTVに反映します。",
      pinTableTitle: "GPIO・シリアル配線",
      colDevice: "モジュール",
      colSignal: "ピン / 設定",
      colTarget: "接続先",
      pinArduino: "Arduino + IR受信機",
      pinArduinoSig: "IR_RECEIVE_PIN 2 · 9600 baud",
      pinArduinoDst: "USBシリアル (/dev/ttyUSB0)",
      pinFndData: "TM1638 FND (DATA)",
      pinFndDataSig: "GPIO 17 (BCM)",
      pinFndClk: "TM1638 FND (CLK)",
      pinFndClkSig: "GPIO 27 (BCM)",
      pinFndStb: "TM1638 FND (STB)",
      pinFndStbSig: "GPIO 22 (BCM)",
      pinFndDst: "Raspberry Pi GPIO (periph.io)",
      btnTableTitle: "短押し / 長押しマップ",
      colHex: "IRコード",
      colShort: "短押し",
      colLong: "長押し",
      colEffect: "動作内容",
      deployTitle: "クロスビルドと systemd 登録",
      deployBody:
        "32bit ARMv6/v7（RPi 1 B+ など）と 64bit ARM 向けにクロスビルドとサービス登録を行うスクリプトを同梱しています。",
      cliTitle: "起動オプション",
      cliBody:
        "-port（既定 /dev/ttyUSB0）、-tv（手動IP指定）、-store（host・MAC・client_key・TLS保存先）、-n（ドライラン）",
      demoTitle: "実機での動作デモ",
      demoLead:
        "ワンボタンでスピーカーと光出力を切り替える様子と、関連リポジトリへのリンクです。",
      demoVideoAlt: "simrmt 実機デモ動画のサムネイル",
      demoWatchCta: "YouTubeで動画を見る",
      demoCardTitle: "ワンボタン音声切替デモ",
      demoCardBody:
        "2023年ブログ記事で公開した実機映像です。同じリモコン操作がそのままGo単一バイナリへ移行しました。",
      linkBlog: "元ブログ記事",
      linkBlogDesc: "LG TV サウンド出力をワンボタンで切り替える（韓国語）",
      linkRepoDesc: "単一Goバイナリとsystemd自動導入スクリプト",
      linkWebosDesc: "PyWebOSTVを移植した純粋Go WebOSクライアント",
      linkDevDesc: "TM1638 8桁7セグメントFNDドライバ",
      footerCopy: "Homin Lee <i@homin.dev> · MIT License",
      footerBackTop: "トップへ戻る"
    },
    zh: {
      docTitle: "simrmt - LG WebOS 电视一键红外遥控器",
      metaDescription:
        "使用闲置红外遥控器、Arduino、树莓派和纯 Go 单一可执行文件，一键切换 LG WebOS 电视音频输出与输入源。",
      navAria: "主导航",
      langAria: "语言切换",
      navStory: "缘起",
      navArch: "架构",
      navHardware: "硬件",
      navDemo: "演示",
      navGithub: "GitHub",
      heroTitle: "切换电视音频输出，<br>只需按一次键",
      heroLead:
        "在 LG WebOS 电视自带扬声器和光纤 5.1 声道功放之间切换时，不再需要层层翻找设置菜单。利用闲置红外遥控器、Arduino、树莓派与纯 Go 单二进制程序即可一键完成。",
      heroCtaGithub: "查看 GitHub",
      heroCtaArch: "查看架构",
      storyTitle: "厌倦深层菜单，<br>索性自己造一个",
      storyP1:
        "把 LG OLED 电视的光纤接口接到用了十几年的 5.1 声道功放上，内置流媒体应用就能输出 5.1 声道音效。但每次在电视扬声器和光纤之间切换，都得进「设置 → 声音」用方向键选择，再按返回退出菜单。",
      storyP2:
        "早先因为 Go 的 WebOS 库不够完善，只能用 Arduino 把红外码发给 Go 串口程序（simremote-ir），再通过本地 HTTP 调用运行 PyWebOSTV 的 Python Flask 服务（simremote）。",
      storyP3:
        "如今将 PyWebOSTV 直接移植为纯 Go 的 go_webostv，去掉了 Python 运行时与 HTTP 中转，合并成单个 Go 程序 simrmt，同时支持 SSDP 自动发现、Wake-on-LAN 和 webOS25 的 wss://:3001 TLS 连接。",
      storyCapSketch: "2023 年在博客中画下的最初构想草图",
      archTitle: "告别双进程中转，<br>合为单个 Go 程序",
      archLead:
        "省去本地 HTTP 与 Python 依赖，直接通过 WebSocket 与 TLS 与电视通信。",
      archLegacyTitle: "旧架构 (simremote-ir + simremote)",
      archLegacyNote: "需要同时维护串口读取的 Go 进程和基于 PyWebOSTV 的 Python Flask 服务。",
      archCurrentTitle: "现架构 (simrmt + go_webostv)",
      archCurrentNote: "串口读取、长短按识别、数码管显示、网络唤醒与电视控制全部集成于单文件。",
      archPreviewAlt: "simrmt 系统架构图预览",
      archInteractiveTitle: "查看源码级交互架构图",
      archInteractiveBody:
        "可点击浏览 main.go、tv.go、fnd.go 与 wol.go 的调用链路与源码位置。",
      archInteractiveCta: "打开交互架构图",
      hwTitle: "硬件接线与<br>按键映射",
      hwLead:
        "Arduino 读取 32 位红外编码并通过 USB 串口发送，simrmt 区分短按与长按后同时更新数码管与电视。",
      pinTableTitle: "引脚连接",
      colDevice: "模块",
      colSignal: "引脚 / 参数",
      colTarget: "连接目标",
      pinArduino: "Arduino + 红外接收器",
      pinArduinoSig: "IR_RECEIVE_PIN 2 · 9600 baud",
      pinArduinoDst: "USB 串口 (/dev/ttyUSB0)",
      pinFndData: "TM1638 数码管 (DATA)",
      pinFndDataSig: "GPIO 17 (BCM)",
      pinFndClk: "TM1638 数码管 (CLK)",
      pinFndClkSig: "GPIO 27 (BCM)",
      pinFndStb: "TM1638 数码管 (STB)",
      pinFndStbSig: "GPIO 22 (BCM)",
      pinFndDst: "树莓派 GPIO (periph.io)",
      btnTableTitle: "短按 / 长按功能表",
      colHex: "红外编码",
      colShort: "短按",
      colLong: "长按",
      colEffect: "执行动作",
      deployTitle: "交叉编译与 systemd 服务部署",
      deployBody:
        "提供一键交叉编译并通过 SSH 部署至 32 位（ARMv6/v7）与 64 位树莓派系统的安装脚本。",
      cliTitle: "运行参数",
      cliBody:
        "-port（默认 /dev/ttyUSB0）、-tv（可选指定 IP）、-store（保存 host / MAC / client_key / TLS）、-n（空跑调试）",
      demoTitle: "实机演示与相关仓库",
      demoLead:
        "用闲置遥控器一键切换电视扬声器与光纤功放的实拍演示视频及相关开源仓库。",
      demoVideoAlt: "simrmt 控制 LG WebOS 电视的演示视频缩略图",
      demoWatchCta: "在 YouTube 观看视频",
      demoCardTitle: "客厅实机一键切换演示",
      demoCardBody:
        "2023 年原博客实拍视频；同样的遥控逻辑如今已全部迁移至纯 Go 单二进制程序。",
      linkBlog: "原博客文章",
      linkBlogDesc: "一键切换 LG 电视音频输出（韩文）",
      linkRepoDesc: "纯 Go 单文件实现与 systemd 安装脚本",
      linkWebosDesc: "移植自 PyWebOSTV 的纯 Go WebOS 客户端",
      linkDevDesc: "TM1638 八位七段数码管驱动库",
      footerCopy: "Homin Lee <i@homin.dev> · MIT License",
      footerBackTop: "返回顶部"
    }
  };

  function normalize(lang) {
    if (!lang) return null;
    var base = String(lang).toLowerCase().split("-")[0];
    return SUPPORTED.indexOf(base) >= 0 ? base : null;
  }

  function detect() {
    try {
      var q = new URLSearchParams(window.location.search).get("lang");
      var normQ = normalize(q);
      if (normQ) return normQ;

      var saved = window.localStorage.getItem(STORAGE_KEY);
      var normSaved = normalize(saved);
      if (normSaved) return normSaved;

      var prefs = Array.isArray(window.navigator.languages) && window.navigator.languages.length
        ? window.navigator.languages
        : [window.navigator.language || "ko"];
      for (var i = 0; i < prefs.length; i++) {
        var normPref = normalize(prefs[i]);
        if (normPref) return normPref;
      }
    } catch (e) {}
    return "ko";
  }

  function t(lang) {
    return strings[normalize(lang) || "ko"];
  }

  return {
    SUPPORTED: SUPPORTED,
    STORAGE_KEY: STORAGE_KEY,
    normalize: normalize,
    detect: detect,
    t: t
  };
})();
