package proxy

import (
	"context"
	"log"
	"sync"
	"time"

	"gopkg.in/mrichman/godnsbl.v1"
)

var (
	DnsBlDomains []string = []string{
		"aspews.ext.sorbs.net",
		"b.barracudacentral.org",
		"bl.deadbeef.com",
		"bl.spamcop.net",
		"blackholes.five-ten-sg.com",
		"blacklist.woody.ch",
		"bogons.cymru.com",
		"cbl.abuseat.org",
		"combined.abuse.ch",
		"combined.rbl.msrbl.net",
		"db.wpbl.info",
		"dnsbl-1.uceprotect.net",
		"dnsbl-2.uceprotect.net",
		"dnsbl-3.uceprotect.net",
		"dnsbl.cyberlogic.net",
		"dnsbl.dronebl.org",
		"dnsbl.inps.de",
		"dnsbl.sorbs.net",
		"drone.abuse.ch",
		"duinv.aupads.org",
		"dul.dnsbl.sorbs.net",
		"dul.ru",
		"dyna.spamrats.com",
		"dynip.rothen.com",
		"http.dnsbl.sorbs.net",
		"images.rbl.msrbl.net",
		"ips.backscatterer.org",
		"ix.dnsbl.manitu.net",
		"korea.services.net",
		"misc.dnsbl.sorbs.net",
		"noptr.spamrats.com",
		"ohps.dnsbl.net.au",
		"omrs.dnsbl.net.au",
		"orvedb.aupads.org",
		"osps.dnsbl.net.au",
		"osrs.dnsbl.net.au",
		"owfs.dnsbl.net.au",
		"owps.dnsbl.net.au",
		"pbl.spamhaus.org",
		"phishing.rbl.msrbl.net",
		"probes.dnsbl.net.au",
		"proxy.bl.gweep.ca",
		"proxy.block.transip.nl",
		"psbl.surriel.com",
		"rdts.dnsbl.net.au",
		"residential.block.transip.nl",
		"ricn.dnsbl.net.au",
		"rmst.dnsbl.net.au",
		"sbl.spamhaus.org",
		"short.rbl.jp",
		"smtp.dnsbl.sorbs.net",
		"socks.dnsbl.sorbs.net",
		"spam.abuse.ch",
		"spam.dnsbl.sorbs.net",
		"spam.rbl.msrbl.net",
		"spam.spamrats.com",
		"spamlist.or.kr",
		"spamrbl.imp.ch",
		"t3direct.dnsbl.net.au",
		"tor.dnsbl.sectoor.de",
		"torserver.tor.dnsbl.sectoor.de",
		"ubl.lashback.com",
		"ubl.unsubscore.com",
		"virbl.bit.nl",
		"virus.rbl.jp",
		"virus.rbl.msrbl.net",
		"web.dnsbl.sorbs.net",
		"wormrbl.imp.ch",
		"xbl.spamhaus.org",
		"zen.spamhaus.org",
		"zombie.dnsbl.sorbs.net",
	}
)

func DnsblChk(ip string) (bool, int) {
	return DnsblChkWithContext(context.Background(), ip)
}

func DnsblChkWithContext(ctx context.Context, ip string) (bool, int) {
	ctx, can := context.WithTimeout(ctx, time.Second*2)
	defer can()

	all := 0
	listed := 0
	var wg sync.WaitGroup

	for _, d := range DnsBlDomains {
		wg.Add(1)

		go func(d string) {
			defer wg.Done()

			res := godnsbl.Lookup(d, ip)
			if len(res.Results) == 0 {
				return
			}
			if res.Results[0].Listed {
				log.Println(res)
				listed++
			}
			all++
		}(d)
	}

	go func() {
		defer can()
		wg.Wait()
		log.Println("done")
	}()

	for {
		select {
		case <-ctx.Done():
			goto done
		default:
			if listed > 6 {
				goto done
			}
		}
	}

done:
	return listed > 6, listed
}
