package geoip

import (
	"fmt"
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

type Region string

func (r Region) String() string {
	return string(r)
}

const (
	EuropeRegion       Region = "EU"
	UnitedStatesRegion Region = "US"
	APACRegion         Region = "APAC"
)

type GeoIP interface {
	Close() (err error)
	Lookup(ip net.IP) GeoInfo
}

type Geo struct {
	countryDB *geoip2.Reader // GeoLite2-Country.mmdb
	asnDB     *geoip2.Reader // GeoLite2-ASN.mmdb
}

func NewGeo(countryPath, asnPath string) (g *Geo, err error) {
	cdb, err := geoip2.Open(countryPath)
	if err != nil {
		return nil, err
	}

	var adb *geoip2.Reader
	if asnPath != "" {
		if adb, err = geoip2.Open(asnPath); err != nil {
			if cErr := cdb.Close(); cErr != nil {
				err = fmt.Errorf("%w, failed to close geoip db: %v", err, cErr)
			}

			return nil, err
		}
	}

	return &Geo{
		countryDB: cdb,
		asnDB:     adb,
	}, nil
}

func (g *Geo) Close() (err error) {
	if g.asnDB != nil {
		if cErr := g.asnDB.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close geoip db: %v", err, cErr)
		}
	}

	if g.countryDB != nil {
		if cErr := g.countryDB.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close geoip db: %v", err, cErr)
		}
	}

	return nil
}

type GeoInfo struct {
	ASN       int
	CC        string // ISO-2
	Continent string // EU, AS, NA, OC, AF, SA, AN
	Region    string
}

func (g *Geo) Lookup(ip net.IP) GeoInfo {
	var out GeoInfo
	if ip == nil {
		out.Region = EuropeRegion.String()
		return out
	}

	if g.asnDB != nil {
		if rec, err := g.asnDB.ASN(ip); err == nil && rec != nil {
			out.ASN = int(rec.AutonomousSystemNumber)
		}
	}

	if g.countryDB != nil {
		if rec, err := g.countryDB.Country(ip); err == nil && rec != nil {
			out.CC = rec.Country.IsoCode
			if rec.Continent.Code != "" {
				out.Continent = rec.Continent.Code
			}
		}
	}

	switch strings.ToUpper(out.Continent) {
	case "US":
		out.Region = UnitedStatesRegion.String()
	case "AS", "OC":
		out.Region = APACRegion.String()
	default:
		out.Region = EuropeRegion.String()
	}

	return out
}
