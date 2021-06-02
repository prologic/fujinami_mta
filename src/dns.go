package proxy

import (
	"errors"
	"log"
	"net"
	"strings"
)

func StripPort(addr net.Addr) string {
	var ip net.IP
	switch addr.(type) {
	case *net.TCPAddr:
		t, _ := addr.(*net.TCPAddr)
		ip = t.IP
	case *net.IPAddr:
		t, _ := addr.(*net.IPAddr)
		ip = t.IP
	default:
		return addr.String()
	}

	return ip.String()
}

func StripEmail(email string) (string, string) {
	email = strings.TrimSpace(email)
	l := strings.Index(email, "@")
	if l < 1 {
		return "", ""
	}
	return email[:l], email[l+1:]
}

func GetMXHosts(domain string) ([]string, error) {
	var result []string

	mxrecords, err := net.LookupMX(domain)
	if err != nil {
		return result, err
	}

	if len(mxrecords) == 0 {
		return result, errors.New("record not found")
	}

	log.Println(mxrecords)
	for _, v := range mxrecords {
		result = append(result, strings.TrimRight(v.Host, "."))
	}

	return result, nil
}

func GetRecord(domain string, ran int) (string, error) {
	mxrecords, err := net.LookupMX(domain)
	if err != nil {
		return "", err
	}

	if len(mxrecords) == 0 {
		return "", errors.New("record not found")
	}

	if len(mxrecords) <= ran {
		return "", errors.New("record not found")
	}

	log.Println(mxrecords)

	return strings.TrimRight(mxrecords[ran].Host, "."), nil
}
