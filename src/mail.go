package proxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/google/uuid"

	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

func readPrivateKey(path string) (*rsa.PrivateKey, error) {
	privateKeyData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	privateKeyBlock, _ := pem.Decode(privateKeyData)
	if privateKeyBlock == nil {
		return nil, errors.New("invalid private key data")
	}
	if privateKeyBlock.Type != "RSA PRIVATE KEY" {
		return nil, errors.New(fmt.Sprintf("invalid private key type : %s", privateKeyBlock.Type))
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, err
}

type Envelove struct {
	From string
	To   string
}

type Header struct {
	Key   string
	Value string
}

type Headers []Header

func (h Headers) Get(label string) (header *Header, ok bool) {
	t := strings.ToLower(label)
	for _, line := range h {
		if strings.ToLower(line.Key) == t {
			return &line, true
		}
	}
	return nil, false
}

func (h Headers) Replace(key, value string) {
	t := strings.ToLower(key)

	for n, line := range h {
		if strings.ToLower(line.Key) == t {
			h[n].Value = value
			break
		}
	}
}

func (h Headers) Prepend(key, value string) {
	h, h[0] = append(h[:1], h[0:]...), Header{key, value}
}

func (h Headers) String() string {
	var s string
	for _, line := range h {
		s += fmt.Sprintf("%s: %s\r\n", line.Key, strings.Trim(line.Value, " \r\n"))
	}
	return s
}

func (h Headers) WriteTo(w io.Writer) (n int, err error) {
	n = 0
	for _, line := range h {
		s, err := fmt.Fprintf(w, "%s: %s\r\n", line.Key, strings.Trim(line.Value, " \r\n"))
		n += s
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

type Mail struct {
	buf     []byte
	body    io.Reader
	Headers Headers
}

func NewMail(body io.Reader) *Mail {
	m := new(Mail)
	var err error

	m.buf, err = ioutil.ReadAll(body)
	if err != nil {
		panic(err)
	}

	return m
}

func (m *Mail) Reader() io.Reader {
	w := new(bytes.Buffer)

	m.Headers.WriteTo(w)
	w.Write([]byte("\r\n"))
	io.Copy(w, m.body)

	return w
}

func (m *Mail) init() {
	reader := bufio.NewReader(bytes.NewReader(m.buf))

	var (
		key   string
		value string
	)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}

		if len(line) == 0 {
			m.Headers = append(m.Headers, Header{key, value})
			break
		}

		if line[0] == ' ' || line[0] == '\t' {
			value += string(line) + "\r\n"
		} else {
			if key != "" {
				m.Headers = append(m.Headers, Header{key, value})
			}

			key = ""
			value = ""
			l := strings.Index(string(line), ":")
			key = string(line[:l])
			value = strings.Trim(string(line[l+1:]), " ")
		}
	}
	if key != "" {
		m.Headers = append(m.Headers, Header{key, value})
	}

	m.body = reader
}

func (m *Mail) Check() error {

	if _, ok := m.Headers.Get("Subject"); !ok {
		return errors.New("Subject Not Found")
	}

	if _, ok := m.Headers.Get("From"); !ok {
		return errors.New("From Not Found")
	}

	if _, ok := m.Headers.Get("To"); !ok {
		return errors.New("To Not Found")
	}

	return nil
}

func NewMessageID(host string) string {
	return fmt.Sprintf("%s@%s", uuid.New().String(), host)
}
