package proxy

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/mileusna/spf"
)

func ParseAddr(addr net.Addr) (net.IP, error) {
	switch addr.(type) {
	case *net.TCPAddr:
		t, _ := addr.(*net.TCPAddr)
		return t.IP, nil
	case *net.IPAddr:
		t, _ := addr.(*net.IPAddr)
		return t.IP, nil
	default:
		return nil, errors.New("Invalid addr")
	}
}

func SpfHeader(addr net.Addr, from string) string {
	var ip net.IP
	ip, err := ParseAddr(addr)
	if err != nil {
		return ""
	}

	_, host := StripEmail(from)
	r := spf.CheckHost(ip, host, from, "")

	return fmt.Sprintf("Authentication-Results: spf=%s ( %s )\r\n", r.String(), from)
}

func ParseAddress(s string) string {
	ss := strings.Split(s, " ")
	return strings.Trim(ss[len(ss)-1], " <>")
}
