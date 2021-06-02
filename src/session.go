package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/emersion/go-smtp"
)

type session struct {
	c    *smtp.Client
	be   *Backend
	st   *smtp.ConnectionState
	opts *smtp.MailOptions
	from string
	to   string
	ctx  context.Context
}

func (s *session) successlog() {
	defer func() {
		recover()
	}()

	log.Printf("200 %s(%s) %s -> %s\r\n", s.st.Hostname, s.st.RemoteAddr, s.from, s.to)
}

func (s *session) Reset() {
	s.from = ""
	s.to = ""
	s.opts = nil
}

func (s *session) Mail(from string, opts smtp.MailOptions) error {
	if s.from != "" {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "Error: nested MAIL command",
		}
	}
	log.Println("MAIL FROM:", from)
	s.from = from
	s.opts = &opts
	return nil
}

func (s *session) Rcpt(to string) error {
	if s.from == "" {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "Error: need MAIL command",
		}
	}

	log.Println("RCPT TO:", to)
	s.to = to

	if err := Allocate(s.ctx, s.from, s.to); err != nil {
		log.Printf("553 %s(%s) %s -> %s\r\n", s.st.Hostname, s.st.RemoteAddr, s.from, s.to)
		return err
	}
	return nil
}

func (s *session) Data(r io.Reader) error {
	if s.to == "" {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "Error: need RCPT command",
		}
	}

	conn, err := s.be.newConn()
	if err != nil {
		return err
	}
	defer conn.Quit()

	conf := GetConfig(s.ctx)

	s.opts.Size = 0
	err = conn.Mail(conf.ProxyEnvelope, s.opts)
	if err != nil {
		return err
	}

	err = conn.Rcpt(conf.ProxyAddress)
	if err != nil {
		return errors.New("Server Error")
	}

	wc, err := conn.Data()
	if err != nil {
		return err
	}

	t := ""
	if s.st.TLS.Version != 0 {
		t = "(version="
		if c, ok := tls_versions[s.st.TLS.Version]; ok {
			t += c
		} else {
			t += fmt.Sprintf("0x%04x", s.st.TLS.Version)
		}

		if c, ok := suites[s.st.TLS.CipherSuite]; ok {
			t += " chiper=" + c
		} else {
			t += fmt.Sprintf(" chiper=0x%04x", s.st.TLS.CipherSuite)
		}
		t += ");\r\n       "
	} else {
		t = ";"
	}

	fmt.Fprintf(wc, "X-Transfer-To: %s\r\n", conf.ProxyAddress)
	fmt.Fprintf(wc, "Deliverd-To: %s\r\n", s.to)

	z := SpfHeader(s.st.RemoteAddr, s.from)
	if z != "" {
		fmt.Fprintf(wc, z)
	}

	fmt.Fprintf(wc, "Received: from %s (%s %s)\r\n"+
		"       by %s (%s %s)\r\n"+
		"       for <%s>"+
		"\r\n       %s%s \r\n",
		s.st.Hostname,
		s.st.Hostname,
		StripPort(s.st.RemoteAddr),
		conf.ServerName,
		StripPort(s.st.LocalAddr),
		"Fujinami SMTP Transfer",
		s.to,
		t,
		now(),
	)

	reader := bufio.NewReader(r)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		if strings.Index(strings.ToLower(string(line)), "message-id") == 0 {
			log.Println(line)
			log.Println(string(line))
			ss := strings.Split(string(line), ":")
			if len(ss) == 2 {
				id := strings.Trim(ss[1], " <>")

				re := new(bytes.Buffer)
				enc := base64.NewEncoder(base64.StdEncoding, re)
				enc.Write([]byte(s.to))
				enc.Close()

				fmt.Fprintf(wc, "Message-ID: <fujinami+%s+%s>\r\n", re.String(), id)
				continue
			}
		}
		wc.Write(line)
		wc.Write([]byte("\r\n"))

		if len(line) == 0 {
			break
		}
	}

	_, err = io.Copy(wc, reader)
	if err != nil {
		log.Println("data writing error:", err)

		wc.Close()
		return err
	}

	err = wc.Close()
	if err != nil {
		log.Println("data writing error:", err)
	}

	s.successlog()
	return nil
}

func (s *session) Logout() error {
	return nil
}

var (
	tls_versions = map[uint16]string{
		tls.VersionTLS10: "TLS1_0",
		tls.VersionTLS11: "TLS1_1",
		tls.VersionTLS12: "TLS1_2",
		tls.VersionTLS13: "TLS1_3",
	}
	suites = map[uint16]string{
		// TLS 1.0 - 1.2 cipher suites.
		tls.TLS_RSA_WITH_RC4_128_SHA:                "TLS_RSA_WITH_RC4_128_SHA",
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:           "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		tls.TLS_RSA_WITH_AES_128_CBC_SHA:            "TLS_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_RSA_WITH_AES_256_CBC_SHA:            "TLS_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256:         "TLS_RSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256:         "TLS_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384:         "TLS_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:        "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:    "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:    "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:          "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:     "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:      "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:      "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:   "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",

		// TLS 1.3 cipher suites.
		tls.TLS_AES_128_GCM_SHA256:       "TLS_AES_128_GCM_SHA256",
		tls.TLS_AES_256_GCM_SHA384:       "TLS_AES_256_GCM_SHA384",
		tls.TLS_CHACHA20_POLY1305_SHA256: "TLS_CHACHA20_POLY1305_SHA256",

		// TLS_FALLBACK_SCSV isn't a standard cipher suite but an indicator
		// that the client is doing version fallback. See RFC 7507.
		tls.TLS_FALLBACK_SCSV: "TLS_FALLBACK_SCSV",
	}
)
