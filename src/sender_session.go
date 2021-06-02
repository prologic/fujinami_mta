package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	n_smtp "net/smtp"
	"strings"

	"github.com/emersion/go-smtp"
)

type sender struct {
	from string
	to   string
	ctx  context.Context
}

func (s *sender) Reset() {
	s.from = ""
	s.to = ""
}

func (s *sender) Mail(from string, opts smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *sender) Rcpt(to string) error {
	s.to = to
	return nil
}

func (s *sender) Data(data io.Reader) error {
	if s.from == "" && s.to == "" {
		return NewBadCommandError()
	}

	_, domain := StripEmail(s.to)
	hosts, err := GetMXHosts(domain)
	if err != nil {
		return NewNotFoundError(s.to)
	}

	conf := GetConfig(s.ctx)

	if !AllowedTo(conf.Allocation, s.from) {
		return NewNotMemberError(s.from)
	}

	from := s.from
	buffer := new(bytes.Buffer)
	reader := bufio.NewReader(data)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		if len(line) == 0 {
			buffer.Write([]byte("\r\n"))
			break
		}
		if strings.Index(strings.ToLower(string(line)), "in-reply-to") == 0 || strings.Index(strings.ToLower(string(line)), "references") == 0 {
			ss := strings.Split(string(line), ":")
			if len(ss) == 2 {
				id := strings.Trim(ss[1], " <>")
				if strings.Index(id, "fujinami+") == 0 {
					spl := strings.Split(id, "+")
					if len(spl) > 3 {
						n, err := base64.StdEncoding.DecodeString(spl[1])
						if err == nil {
							from = string(n)
							fmt.Fprintf(buffer, "%s: %s\r\n", ss[0], spl[2])
							continue
						}
					}
				}
			}
		}
		if strings.Index(strings.ToLower(string(line)), "from") == 0 {
			fmt.Fprintf(buffer, "From: %s <%s>\r\n", conf.FromName, from)
			continue
		}
		buffer.Write(line)
		buffer.Write([]byte("\r\n"))
	}

	io.Copy(buffer, reader)

	d, _ := ioutil.ReadAll(buffer)

	for _, host := range hosts {
		err = n_smtp.SendMail(host+":smtp", nil, from, []string{s.to}, d)
		if err == nil {
			log.Printf("[send] %s -> %s", from, s.to)
			return nil
		}
	}

	return NewError(err)
}

func (s *sender) Logout() error {
	return nil
}
