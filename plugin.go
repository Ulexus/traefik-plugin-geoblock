package traefik_plugin_geoblock

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"go/build"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ip2location/ip2location-go/v9"
)

type Config struct {
	Enabled          bool
	DatabaseFilePath string
	AllowedCountries []string
	AllowPrivate     bool
}

func CreateConfig() *Config {
	return &Config{}
}

type Plugin struct {
	next http.Handler
	name string
	db   *ip2location.DB
	cfg  *Config
}

func New(_ context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	db, err := ip2location.OpenDB(cfg.DatabaseFilePath)
	if err != nil {
		cwd, _ := os.Getwd()
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			gopath = build.Default.GOPATH
		}
		return nil, fmt.Errorf("failed to open database: %w (cwd: %s, gopath: %s, env: %#v, loc: %s)", err, cwd, gopath, os.Environ(), string(modLocation))
	}

	return &Plugin{
		next: next,
		name: name,
		db:   db,
		cfg:  cfg,
	}, nil
}

func (p Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !p.cfg.Enabled {
		p.next.ServeHTTP(rw, req)
		return
	}

	ips := p.GetRemoteIPs(req)

	for _, ip := range ips {
		country, err := p.CheckAllowed(ip)
		if err != nil {
			if errors.Is(err, ErrNotAllowed) {
				log.Printf("%s: %s - access denied for %s (%s)", p.name, req.Host, ip, country)
				rw.WriteHeader(http.StatusForbidden)
				return
			} else {
				log.Printf("%s: %s - %v", p.name, req.Host, err)
				rw.WriteHeader(http.StatusForbidden)
				return
			}
		}
	}

	p.next.ServeHTTP(rw, req)
}

// GetRemoteIPs collects the remote IPs from the X-Forwarded-For and X-Real-IP headers.
func (p Plugin) GetRemoteIPs(req *http.Request) (ips []string) {
	ipMap := make(map[string]struct{})

	if xff := req.Header.Get("x-forwarded-for"); xff != "" {
		for _, ip := range strings.Split(xff, ",") {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			ipMap[ip] = struct{}{}
		}
	}
	if xri := req.Header.Get("x-real-ip"); xri != "" {
		for _, ip := range strings.Split(xri, ",") {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			ipMap[ip] = struct{}{}
		}
	}

	for ip := range ipMap {
		ips = append(ips, ip)
	}

	return
}

var ErrNotAllowed = errors.New("not allowed")

// CheckAllowed checks whether a given IP address is allowed according to the configured allowed countries.
func (p Plugin) CheckAllowed(ip string) (string, error) {
	country, err := p.Lookup(ip)
	if err != nil {
		return "", fmt.Errorf("lookup of %s failed: %w", ip, err)
	}

	if country == "-" { // Private address
		if p.cfg.AllowPrivate {
			return country, nil
		}
		return country, ErrNotAllowed
	}

	var allowed bool
	for _, allowedCountry := range p.cfg.AllowedCountries {
		if allowedCountry == country {
			allowed = true
			break
		}
	}
	if !allowed {
		return country, ErrNotAllowed
	}

	return country, nil
}

// Lookup queries the ip2location database for a given IP address.
func (p Plugin) Lookup(ip string) (string, error) {
	record, err := p.db.Get_country_short(ip)
	if err != nil {
		return "", err
	}

	country := record.Country_short
	if strings.HasPrefix(strings.ToLower(country), "invalid") {
		return "", errors.New(country)
	}

	return record.Country_short, nil
}
