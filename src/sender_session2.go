package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	n_smtp "net/smtp"
	"strings"

	"github.com/emersion/go-msgauth/dkim"
	"github.com/emersion/go-smtp"

	"gosmtp/src/store"
)

type sender2 struct {
	from string
	to   string
	st   *smtp.ConnectionState
	ctx  context.Context
}

func (s *sender2) Reset() {
	s.from = ""
	s.to = ""
}

func (s *sender2) Mail(from string, opts smtp.MailOptions) error {
	if s.from != "" {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "Error: nested MAIL command",
		}
	}
	s.from = from

	log.Println("MAIL FROM:", from)
	return nil
}

func (s *sender2) Rcpt(to string) error {
	if s.from == "" {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "Error: need MAIL command",
		}
	}
	s.to = to

	conf := GetConfig(s.ctx)

	if !AllowedTo(conf.Allocation, s.from) {
		return NewNotMemberError(s.from)
	}

	log.Println("RCPT TO:", to)
	return nil
}

func (s *sender2) Data(data io.Reader) error {
	if s.from == "" && s.to == "" {
		return NewBadCommandError()
	}

	from := ""

	m := NewMail(data)
	m.init()
	err := m.Check()
	if err != nil {
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      err.Error(),
		}
	}

	tr, _ := m.Headers.Get("To")
	tos := strings.Split(tr.Value, ",")
	to := ParseAddress(tos[0])
	f, _ := m.Headers.Get("From")
	fr := ParseAddress(f.Value)

	if fro, ok := store.Current().Get(to); ok {
		from = fro
	} else {
		_, dom := StripEmail(fr)
		from = fmt.Sprintf("si-%s@%s", RandString1(4), dom)

		store.Current().Set(to, from)
	}

	conf := GetConfig(s.ctx)
	if from != "" {
		m.Headers.Replace("From", conf.FromName+" <"+from+">")
	}

	t := ""
	if s.st.TLS.Version != 0 {
		t = "\r\n       (version="
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

	m.Headers.Prepend(
		"Received",
		fmt.Sprintf(
			"from %s (%s)\r\n"+
				"       by: %s (%s %s)\r\n"+
				"       for: <%s>%s%s",
			s.st.Hostname,
			conf.ServerName,
			StripPort(s.st.RemoteAddr),
			StripPort(s.st.LocalAddr),
			conf.Name,
			s.to,
			t,
			now(),
		),
	)

	privateKey, _ := readPrivateKey(conf.DkimPrivate)

	options := &dkim.SignOptions{
		Domain:   conf.DkimDomain,
		Selector: conf.DkimSelector,
		Signer:   privateKey,
	}

	var b bytes.Buffer
	if err := dkim.Sign(&b, m.Reader(), options); err != nil {
		log.Fatal(err)
	}

	z, _ := ioutil.ReadAll(&b)

	_, domain := StripEmail(s.to)
	hosts, err := GetMXHosts(domain)
	if err != nil {
		return NewNotFoundError(s.to)
	}

	for _, host := range hosts {
		err = n_smtp.SendMail(host+":smtp", nil, from, []string{s.to}, z)
		if err == nil {
			log.Printf("200 %s(%s) %s -> %s\r\n", s.st.Hostname, s.st.RemoteAddr, s.from, s.to)
			return nil
		}
	}

	return NewError(err)
}

func (s *sender2) Logout() error {
	return nil
}
