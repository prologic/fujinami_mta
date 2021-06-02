package proxy

import (
	"time"
)

const (
	format string = "Mon, 02 Jan 2006 15:04:05 -0700 (MST)"
)

var (
	jst *time.Location = time.FixedZone("Asia/Tokyo", 9*60*60)
)

func now() string {
	now := time.Now()
	now.In(jst)
	return now.Format(format)
}
