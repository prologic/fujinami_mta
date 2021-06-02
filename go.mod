module gosmtp

go 1.13

require (
	github.com/emersion/go-msgauth v0.6.5
	github.com/emersion/go-smtp v0.12.0
	github.com/emersion/go-smtp-proxy v0.0.0-20200210193521-e8e7dd723514
	github.com/google/uuid v1.2.0
	github.com/miekg/dns v1.1.42 // indirect
	github.com/mileusna/spf v0.9.3
	github.com/prologic/bitcask v0.3.10
	golang.org/x/crypto v0.0.0-20210506145944-38f3c27a63bf // indirect
	golang.org/x/net v0.0.0-20210508051633-16afe75a6701 // indirect
	golang.org/x/sys v0.0.0-20210507161434-a76c4d0a0096 // indirect
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.10
)

replace github.com/emersion/go-smtp => ../go-smtp
