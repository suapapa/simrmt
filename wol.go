package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

const (
	arpTablePath = "/proc/net/arp"
	wolPort      = 9
)

func buildMagicPacket(macAddr string) ([]byte, error) {
	hw, err := net.ParseMAC(strings.TrimSpace(macAddr))
	if err != nil {
		return nil, fmt.Errorf("parse MAC %q: %w", macAddr, err)
	}
	if len(hw) != 6 {
		return nil, fmt.Errorf("unexpected MAC length %d for %q", len(hw), macAddr)
	}

	packet := make([]byte, 6+16*len(hw))
	for i := range 6 {
		packet[i] = 0xff
	}
	for i := range 16 {
		copy(packet[6+i*len(hw):], hw)
	}
	return packet, nil
}

func sendWakeOnLAN(macAddr string) error {
	packet, err := buildMagicPacket(macAddr)
	if err != nil {
		return err
	}

	dst := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: wolPort,
	}
	conn, err := net.DialUDP("udp4", nil, dst)
	if err != nil {
		return fmt.Errorf("dial UDP broadcast: %w", err)
	}
	defer conn.Close()

	n, err := conn.Write(packet)
	if err != nil {
		return fmt.Errorf("send magic packet: %w", err)
	}
	if n != len(packet) {
		return fmt.Errorf("short write sending magic packet: %d/%d", n, len(packet))
	}
	return nil
}

func resolveMAC(ip string) (string, error) {
	if ip == "" {
		return "", errors.New("empty IP address")
	}

	if data, err := os.ReadFile(arpTablePath); err == nil {
		if mac, err := parseARPTable(ip, string(data)); err == nil {
			return mac, nil
		}
	}

	out, err := exec.Command("arp", "-n", ip).Output()
	if err != nil {
		return "", fmt.Errorf("arp lookup for %q: %w", ip, err)
	}
	return parseARPTable(ip, string(out))
}

func parseARPTable(ip, table string) (string, error) {
	parenIP := "(" + ip + ")"
	for _, line := range strings.Split(table, "\n") {
		fields := strings.Fields(line)
		if !lineMatchesIP(fields, ip, parenIP) {
			continue
		}
		for _, field := range fields {
			if mac, ok := parseMACField(field); ok {
				return mac, nil
			}
		}
	}
	return "", fmt.Errorf("MAC address not found in ARP table for %q", ip)
}

func lineMatchesIP(fields []string, ip, parenIP string) bool {
	for _, f := range fields {
		if f == ip || f == parenIP {
			return true
		}
	}
	return false
}

func parseMACField(field string) (string, bool) {
	normalized := normalizeColonMAC(field)
	hw, err := net.ParseMAC(normalized)
	if err != nil || len(hw) != 6 {
		return "", false
	}
	mac := hw.String()
	if mac == "00:00:00:00:00:00" {
		return "", false
	}
	return mac, true
}

// normalizeColonMAC pads single-digit octets produced by BSD/macOS arp (e.g., "a:bb:..." -> "0a:bb:...").
func normalizeColonMAC(s string) string {
	parts := strings.Split(s, ":")
	if len(parts) != 6 {
		return s
	}
	for i, part := range parts {
		if len(part) == 1 {
			parts[i] = "0" + part
		}
	}
	return strings.Join(parts, ":")
}
