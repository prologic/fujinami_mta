package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"

	"github.com/emersion/go-smtp"
)

type Security int

const (
	SecurityTLS Security = iota
	SecurityStartTLS
	SecurityNone
)

type Backend struct {
	Addr      string
	Security  Security
	TLSConfig *tls.Config
	LMTP      bool
	Host      string
	Config    *Config

	unexported struct{}
	BaseCtx    func() context.Context
	baseCtx    context.Context
}

func New(addr string, conf *Config) *Backend {
	return &Backend{Addr: addr, Security: SecurityStartTLS, Config: conf}
}

func NewTLS(addr string, tlsConfig *tls.Config) *Backend {
	return &Backend{
		Addr:      addr,
		Security:  SecurityTLS,
		TLSConfig: tlsConfig,
	}
}

func NewLMTP(addr string, host string) *Backend {
	return &Backend{
		Addr:     addr,
		Security: SecurityNone,
		LMTP:     true,
		Host:     host,
	}
}

func (be *Backend) Context() context.Context {
	if be.baseCtx == nil {
		if be.BaseCtx != nil {
			be.baseCtx = be.BaseCtx()
		}
		be.baseCtx = SetConfig(be.baseCtx, be.Config)
	}
	return be.baseCtx
}

func (be *Backend) newConn() (*smtp.Client, error) {
	var conn net.Conn
	var err error
	if be.LMTP {
		if be.Security != SecurityNone {
			return nil, errors.New("smtp-proxy: LMTP doesn't support TLS")
		}
		conn, err = net.Dial("unix", be.Addr)
	} else if be.Security == SecurityTLS {
		conn, err = tls.Dial("tcp", be.Addr, be.TLSConfig)
	} else {
		conn, err = net.Dial("tcp", be.Addr)
	}
	if err != nil {
		return nil, err
	}

	var c *smtp.Client
	if be.LMTP {
		c, err = smtp.NewClientLMTP(conn, be.Host)
	} else {
		host := be.Host
		if host == "" {
			host, _, _ = net.SplitHostPort(be.Addr)
		}
		c, err = smtp.NewClient(conn, host)
	}
	if err != nil {
		return nil, err
	}

	if be.Security == SecurityStartTLS {
		if err := c.StartTLS(be.TLSConfig); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (be *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	log.Println("[login]", username, "from", state.RemoteAddr)
	for _, usr := range be.Config.Users {
		if usr.Name == username && usr.PlainPassword == password {
			log.Println("[login] success")
			return &sender2{
				st:  state,
				ctx: be.Context(),
			}, nil
		}
	}

	return nil, errors.New("Invalid username or password")
}

func (be *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	log.Println("[AnonymousLogin] HELO", state.Hostname)
	s := &session2{
		be:  be,
		st:  state,
		ctx: be.Context(),
	}

	return s, nil
}
