package daemon

import (
	"net"
	"strconv"
	"strings"
)

type Config struct {
	HostHTTPAddr        string
	ParticipantHTTPAddr string
	ParticipantURL      string
	OpenBrowser         bool
	Mock                bool
}

func LoadConfig(hostAddr, participantAddr string, mock, openBrowser bool) Config {
	return Config{
		HostHTTPAddr:        hostAddr,
		ParticipantHTTPAddr: participantAddr,
		ParticipantURL:      participantURL(participantAddr),
		OpenBrowser:         openBrowser,
		Mock:                mock,
	}
}

func participantURL(bindAddr string) string {
	host, port, err := net.SplitHostPort(bindAddr)
	if err != nil {
		return "http://" + bindAddr + "/"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = lanIPv4()
		if host == "" {
			host = "127.0.0.1"
		}
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}

func lanIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			ip = ip.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func hostURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr + "/"
	}
	return "http://" + addr + "/"
}

func listenPort(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return port
}

func formatAddrLabel(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" {
		host = "0.0.0.0"
	}
	if p, err := strconv.Atoi(port); err == nil && p == 80 {
		return host
	}
	return net.JoinHostPort(host, port)
}
