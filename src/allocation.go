package proxy

import (
	"context"
	"log"
	"strings"
)

func Allocate(ctx context.Context, from, to string) error {
	// TODO: add lower/upper check
	conf := GetConfig(ctx)

	if ok := AllowedTo(conf.Allocation, to); !ok {
		log.Printf("[at] deny to: %s from: %s\n", to, from)
		return NewNotMemberError(to)
	}

	if ok := AllowedFrom(conf.Allocation, from); !ok {
		log.Printf("[af] deny to: %s from: %s\n", to, from)
		return NewNotMemberError(to)
	}

	return nil
}

func AllowedFrom(a AllocationSetting, from string) bool {
	_, host := StripEmail(from)
	if len(host) == 0 {
		return false
	}

	_, ok := a.BlacklistHosts[strings.ToLower(host)]
	return !ok
}

func AllowedTo(a AllocationSetting, to string) bool {
	_, host := StripEmail(to)
	if len(host) == 0 {
		return false
	}

	if ad, ok := a.ToAddresses[host]; ok {
		return ad
	}

	if ad, ok := a.ToDomains[host]; ok {
		return ad
	}

	return false
}
