package main

import (
	"bufio"
	"fmt"
	"github.com/go-resty/resty/v2"
	"go4.org/netipx"
	"io"
	"log"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

type Rdap struct {
	Description string        `json:"description"`
	Publication string        `json:"publication"`
	Services    [][2][]string `json:"services"`
}

type IPv6 struct {
	RIR string
	Ipp netip.Prefix
}

type IPv4 struct {
	RIR string
	Ipp netip.Prefix
}

type ASN struct {
	RIR   string
	Start int
	End   int
}

type Iana struct {
	v4s []IPv4
	v6s []IPv6
	asn []ASN
}

func (obj Iana) LookUPWithASN(value int) (string, bool) {
	for _, v := range obj.asn {
		if v.Start <= value && v.End >= value {
			return v.RIR, true
		}
	}

	return "", false
}

func (obj Iana) LookUP(ipv netip.Addr) (string, bool) {

	if ipv.Is4() {
		for _, v := range obj.v4s {
			if v.Ipp.Contains(ipv) {
				return v.RIR, true
			}
		}
	} else if ipv.Is6() {
		for _, v := range obj.v6s {
			if v.Ipp.Contains(ipv) {
				return v.RIR, true
			}
		}
	}

	return "", false
}

type Transfer struct {
	FromORG string
	FromISO string
	FromRIR string
	ToORG   string
	ToISO   string
	ToRIR   string
	ToDate  string
}

type IPTransfer struct {
	Ipp netip.Prefix
	Transfer
}

type ASTransfer struct {
	ASN int
	Transfer
}

func main() {

	v4m := make(map[string]IPTransfer)
	v6m := make(map[string]IPTransfer)

	lacnicRIR := netipx.IPSetBuilder{}
	ripeRIR := netipx.IPSetBuilder{}
	apnicRIR := netipx.IPSetBuilder{}
	arinRIR := netipx.IPSetBuilder{}
	afrinicRIR := netipx.IPSetBuilder{}

	ips := downIPv4()
	for _, v := range ips {
		switch strings.ToUpper(v.RIR) {
		case "APNIC":
			apnicRIR.AddPrefix(v.Ipp)
		case "RIPE":
			ripeRIR.AddPrefix(v.Ipp)
		case "LACNIC":
			lacnicRIR.AddPrefix(v.Ipp)
		case "ARIN":
			arinRIR.AddPrefix(v.Ipp)
		case "AFRINIC":
			afrinicRIR.AddPrefix(v.Ipp)
		}
	}

	dst := "c:/study/transfer.log"
	//res, err := resty.New().R().SetOutput(dst).Get("https://ftp.apnic.net/transfers-all/apnic/transfer-all-apnic-latest")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Println(res.StatusCode())
	fr, _ := os.Open(dst)
	rr := bufio.NewReader(fr)
	for {
		line, _, e := rr.ReadLine()
		if e == io.EOF {
			break
		} else if e != nil {
			log.Fatal(e)
		}

		str := string(line)
		if strings.HasPrefix(str, "#") {
			continue
		}
		tmp := strings.Split(str, "|")
		if len(tmp) < 10 {
			continue
		}

		value := tmp[1]

		tt := Transfer{
			FromORG: tmp[2],
			FromISO: tmp[3],
			FromRIR: tmp[4],
			ToORG:   tmp[6],
			ToISO:   tmp[7],
			ToRIR:   tmp[8],
			ToDate:  tmp[9],
		}

		switch tmp[0] {
		case "asn":
			at := ASTransfer{
				Transfer: tt,
			}
			at.ASN, _ = strconv.Atoi(value)
		case "ipv4":
			it := IPTransfer{
				Ipp:      netip.MustParsePrefix(value),
				Transfer: tt,
			}
			v4m[value] = it

			switch tt.FromRIR {
			case "APNIC":
				apnicRIR.RemovePrefix(it.Ipp)
			case "RIPE":
				ripeRIR.RemovePrefix(it.Ipp)
			case "LACNIC":
				lacnicRIR.RemovePrefix(it.Ipp)
			case "ARIN":
				arinRIR.RemovePrefix(it.Ipp)
			case "AFRINIC":
				afrinicRIR.RemovePrefix(it.Ipp)
			}
			switch tt.ToRIR {
			case "APNIC":
				apnicRIR.AddPrefix(it.Ipp)
			case "RIPE":
				ripeRIR.AddPrefix(it.Ipp)
			case "LACNIC":
				lacnicRIR.AddPrefix(it.Ipp)
				fmt.Println(it)
			case "ARIN":
				arinRIR.AddPrefix(it.Ipp)
			case "AFRINIC":
				afrinicRIR.AddPrefix(it.Ipp)
			}
		case "ipv6":
			v6m[value] = IPTransfer{
				Ipp:      netip.MustParsePrefix(value),
				Transfer: tt,
			}
		}
	}

	apnicIPs, e := apnicRIR.IPSet()
	if e == nil {
		fmt.Println("apnic", len(apnicIPs.Prefixes()))
	}
	lacnicIPs, e := lacnicRIR.IPSet()
	if e == nil {
		fmt.Println("lacnic", len(lacnicIPs.Prefixes()))
	}
	arinIPs, e := arinRIR.IPSet()
	if e == nil {
		fmt.Println("arin", len(arinIPs.Prefixes()))
	}
	ripeIPs, e := ripeRIR.IPSet()
	if e == nil {
		fmt.Println("ripe", len(ripeIPs.Prefixes()))
	}
	afrinicIPs, e := afrinicRIR.IPSet()
	if e == nil {
		fmt.Println("afrinic", len(afrinicIPs.Prefixes()))
		for _, ipp := range afrinicIPs.Prefixes() {
			fmt.Println(ipp)
		}
	}
	return
	for k, tt := range v4m {

		ipp := netip.MustParsePrefix(k)
		bit := ipp.Bits()
		ipn := netipx.PrefixIPNet(ipp)
		for start := bit - 1; start >= 8; start -= 1 {
			msk := net.CIDRMask(start, 32)
			ipn.IP.Mask(msk)
			ipn.Mask = msk

			key := ipn.String()

			if vv, ok := v4m[key]; ok {
				fmt.Println(k, key)
				if vv.ToDate > tt.ToDate {
					fmt.Println(tt.ToRIR, tt.ToDate)
					fmt.Println(vv.ToRIR, vv.ToDate)
				}
				if tt.ToDate == vv.ToDate {
					fmt.Println(tt.ToRIR, tt.ToDate)
					fmt.Println(vv.ToRIR, vv.ToDate)
				} else {

				}

				fmt.Println()
			}
		}
	}

	for k, tt := range v6m {

		ipp := netip.MustParsePrefix(k)
		bit := ipp.Bits()
		ipn := netipx.PrefixIPNet(ipp)
		for start := bit - 1; start >= 8; start -= 1 {
			msk := net.CIDRMask(start, 128)
			ipn.IP.Mask(msk)
			ipn.Mask = msk

			key := ipn.String()

			if vv, ok := v6m[key]; ok {
				fmt.Println(k, key)
				fmt.Println(tt.ToRIR)
				fmt.Println(vv.ToRIR)
				fmt.Println()
			}
		}
	}

	return
	obj := NewIana()

	fmt.Println(obj.LookUP(netip.MustParseAddr("8.9.8.8")))
	fmt.Println(obj.LookUP(netip.MustParseAddr("188.9.8.8")))
}

func NewIana() *Iana {
	return &Iana{
		v4s: downIPv4(),
		v6s: downIPv6(),
		asn: downASN(),
	}
}

func downIPv6() []IPv6 {
	var m Rdap

	resty.New().R().SetResult(&m).Get("https://data.iana.org/rdap/ipv6.json")

	//fmt.Println(m)
	data := make([]IPv6, 0, 10000)
	for _, record := range m.Services {
		vs := record[0]
		hs := record[1]

		rir, ok := rirSource(hs)
		if ok {
			for _, v := range vs {
				ipp := netip.MustParsePrefix(v)

				data = append(data, IPv6{
					RIR: rir,
					Ipp: ipp,
				})
			}
		}
	}

	return data
}

func downIPv4() []IPv4 {
	var m Rdap

	resty.New().R().SetResult(&m).Get("https://data.iana.org/rdap/ipv4.json")

	//fmt.Println(m)
	data := make([]IPv4, 0, 10000)
	for _, record := range m.Services {
		vs := record[0]
		hs := record[1]

		rir, ok := rirSource(hs)
		if ok {
			for _, v := range vs {
				ipp := netip.MustParsePrefix(v)

				data = append(data, IPv4{
					RIR: rir,
					Ipp: ipp,
				})
			}
		}
	}

	return data
}

func downASN() []ASN {

	var m Rdap

	resty.New().R().SetResult(&m).Get("https://data.iana.org/rdap/asn.json")

	//fmt.Println(m)

	var data = make([]ASN, 0, 10000)

	for _, record := range m.Services {
		vs := record[0]
		hs := record[1]

		rir, ok := rirSource(hs)
		if ok {
			for _, v := range vs {
				a := strings.Split(v, "-")
				if len(a) > 1 {
					A := ASN{
						RIR: rir,
					}
					A.Start, _ = strconv.Atoi(a[0])
					A.End, _ = strconv.Atoi(a[1])

					data = append(data, A)
				}
			}
		}
	}

	return data
}

func rirSource(hs []string) (string, bool) {
	if strings.Contains(hs[0], "afrinic") {
		return "afrinic", true
	} else if strings.Contains(hs[0], "apnic") {
		return "apnic", true
	} else if strings.Contains(hs[0], "lacnic") {
		return "lacnic", true
	} else if strings.Contains(hs[0], "ripe") {
		return "ripe", true
	} else if strings.Contains(hs[0], "arin") {
		return "arin", true
	} else {
		return "", false
	}
}
