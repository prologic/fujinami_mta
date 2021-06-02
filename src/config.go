package proxy

import (
	"context"
	"crypto/tls"
)

const ConfigKey string = "_config"

func SetConfig(ctx context.Context, conf *Config) context.Context {
	return context.WithValue(ctx, ConfigKey, conf)
}

func GetConfig(ctx context.Context) *Config {
	c, ok := ctx.Value(ConfigKey).(*Config)
	if !ok {
		panic("config not found")
	}
	return c
}

type Config struct {
	Name          string
	ServerName    string
	Users         []User
	Listeners     Listener
	ListenDomains []ListenDomain
	ProxyAddress  string
	ProxyEnvelope string
	Allocation    AllocationSetting
	FromName      string
	DkimSelector  string
	DkimPrivate   string
	DkimDomain    string
}

type AllocationSetting struct {
	ToAddresses    map[string]bool
	ToDomains      map[string]bool
	BlacklistHosts map[string]bool
}

type User struct {
	Name          string
	PlainPassword string
}

type Listener struct {
	Port         int
	TlsConfig    *tls.Config
	ReadTimeout  int
	WriteTimeout int
}

type ListenDomain string
