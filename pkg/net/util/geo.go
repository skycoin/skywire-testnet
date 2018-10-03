package util

import (
	"github.com/oschwald/maxminddb-golang"
	"net"
)

var IPLocator = &ipLocator{}

type ipLocator struct {
	db *maxminddb.Reader
	ok bool
}

func (l *ipLocator) Init(file string) (err error) {
	l.db, err = maxminddb.Open(file)
	if err != nil {
		return
	}
	l.ok = true
	return
}

func (l *ipLocator) IsOK() bool {
	return l.ok
}

func (l *ipLocator) Close() {
	l.db.Close()
}

type record struct {
	Country struct {
		Nams struct {
			Name string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"country"`
	Subs []struct {
		Nams struct {
			Name string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

func (l *ipLocator) LookupLocation(ip string) (location string) {
	t := net.ParseIP(ip)
	if t == nil {
		return
	}
	r := &record{}
	err := l.db.Lookup(t, r)
	if err != nil {
		return
	}
	if len(r.Subs) > 0 && len(r.Subs[0].Nams.Name) > 0 {
		location = r.Subs[0].Nams.Name + ", "
	}
	location += r.Country.Nams.Name
	return
}
