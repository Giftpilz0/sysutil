package netinfo

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}

var (
	IPInfoURL    = "https://ipinfo.io/json"
	MACLookupURL = "https://api.maclookup.app/v2/macs/"
)

type IPInfo struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname,omitempty"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Location string `json:"loc"`
	Org      string `json:"org"`
	Timezone string `json:"timezone"`
}

type MACInfo struct {
	Found     bool   `json:"found"`
	MACPrefix string `json:"macPrefix"`
	Company   string `json:"company"`
	Address   string `json:"address"`
	Country   string `json:"country"`
	BlockType string `json:"blockType"`
	IsRandom  bool   `json:"isRand"`
	IsPrivate bool   `json:"isPrivate"`
}

type Interface struct {
	Name string
	MAC  string
	Up   bool
}

func LookupIP(ctx context.Context) (IPInfo, error) {
	var info IPInfo
	err := getJSON(ctx, IPInfoURL, &info)

	return info, err
}

func LookupMAC(ctx context.Context, mac string) (MACInfo, error) {
	normalized, err := NormalizeMAC(mac)
	if err != nil {
		return MACInfo{}, err
	}

	var info MACInfo
	if err := getJSON(ctx, MACLookupURL+normalized, &info); err != nil {
		return MACInfo{}, err
	}

	return info, nil
}

// NormalizeMAC requires at least the 3 byte vendor prefix.
func NormalizeMAC(mac string) (string, error) {
	s := strings.NewReplacer(":", "", "-", "", ".", "", " ", "").Replace(strings.TrimSpace(mac))
	if len(s) < 6 || len(s) > 12 {
		return "", fmt.Errorf("invalid MAC address %q: need between 6 and 12 hex digits", mac)
	}

	if _, err := hex.DecodeString(s[:len(s)&^1]); err != nil {
		return "", fmt.Errorf("invalid MAC address %q: not hexadecimal", mac)
	}

	return strings.ToUpper(s), nil
}

func Interfaces() ([]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var out []Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		out = append(out, Interface{
			Name: iface.Name,
			MAC:  iface.HardwareAddr.String(),
			Up:   iface.Flags&net.FlagUp != 0,
		})
	}

	return out, nil
}

// DefaultMAC prefers the first interface that is up.
func DefaultMAC() (Interface, error) {
	ifaces, err := Interfaces()
	if err != nil {
		return Interface{}, err
	}

	if len(ifaces) == 0 {
		return Interface{}, errors.New("no network interface with a MAC address found")
	}

	for _, iface := range ifaces {
		if iface.Up {
			return iface, nil
		}
	}

	return ifaces[0], nil
}

func getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error   any    `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &apiErr)

		msg := apiErr.Message
		if msg == "" && apiErr.Error != nil {
			msg = fmt.Sprint(apiErr.Error)
		}
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}

		return fmt.Errorf("%s: %s (HTTP %d)", req.URL.Host, msg, resp.StatusCode)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decoding response from %s: %w", req.URL.Host, err)
	}

	return nil
}
