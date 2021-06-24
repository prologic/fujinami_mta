package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/mileusna/spf"

	"gosmtp/src/store"
)

type session2 struct {
	c         *smtp.Client
	be        *Backend
	st        *smtp.ConnectionState
	opts      *smtp.MailOptions
	from      string
	to        string
	spfResult spf.Result
	ctx       context.Context
}

func (s *session2) successlog() {
	defer func() {
		recover()
	}()

	log.Printf("200 %s(%s) %s -> %s\r\n", s.st.Hostname, s.st.RemoteAddr, s.from, s.to)
}

func (s *session2) Reset() {
	s.from = ""
	s.to = ""
	s.opts = nil
}

func (s *session2) Mail(from string, opts smtp.MailOptions) error {
	if s.from != "" {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "Error: nested MAIL command",
		}
	}

	if from == "" {
		log.Printf("501 %s(%s) %s -> %s\r\n", s.st.Hostname, s.st.RemoteAddr, s.from, s.to)
		return &smtp.SMTPError{
			Code:         501,
			EnhancedCode: smtp.EnhancedCode{5, 0, 1},
			Message:      "Error: Do not Empty",
			ForceClose:   true,
		}
	}
	log.Println("MAIL FROM:", from)
	s.from = from
	s.opts = &opts

	ip, _ := ParseAddr(s.st.RemoteAddr)
	s.spfResult = spf.CheckHost(ip, s.st.Hostname, from, s.st.Hostname)

	return nil
}

func (s *session2) Rcpt(to string) error {
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

func (s *session2) Data(r io.Reader) error {
	if s.to == "" {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "Error: need RCPT command",
		}
	}

	black, blcnt := DnsblChkWithContext(s.ctx, StripPort(s.st.RemoteAddr))
	if black {
		log.Printf("503 %s(%s) %s -> %s\r\n", s.st.Hostname, s.st.RemoteAddr, s.from, s.to)
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 7, 0},
			Message:      "Error: You are in too many blacklists.",
			ForceClose:   true,
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

	fmt.Fprintf(wc, "X-Blacklist-Count: %d (%s)\r\n", blcnt, StripPort(s.st.RemoteAddr))
	fmt.Fprintf(wc, "Return-Path: <%s>\r\n", s.from)
	fmt.Fprintf(wc, "X-Transfer-To: <%s>\r\n", conf.ProxyAddress)
	fmt.Fprintf(wc, "Deliverd-To: <%s>\r\n", s.to)

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
		conf.Name,
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
		if strings.Index(strings.ToLower(string(line)), "from") == 0 {
			ss := strings.Split(string(line), ":")
			if len(ss) == 2 {
				from := ParseAddress(ss[1])
				store.Current().Set(from, s.to)
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
		log.Println("data Coping error:", err)

		wc.Close()
		return err
	}

	err = wc.Close()
	if err != nil {
		log.Println("data Closing error:", err)
		return err
	}

	s.successlog()
	return nil
}

func (s *session2) Logout() error {
	return nil
}
