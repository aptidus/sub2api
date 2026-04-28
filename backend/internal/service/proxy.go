package service

import (
	"net"
	"net/url"
	"strconv"
	"time"
)

type Proxy struct {
	ID        int64
	Name      string
	Protocol  string
	Host      string
	Port      int
	Username  string
	Password  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Proxy) IsActive() bool {
	return p.Status == StatusActive
}

func (p *Proxy) URL() string {
	u := &url.URL{
		Scheme: p.Protocol,
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
	}
	if p.Username != "" && p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u.String()
}

func (p *Proxy) MaskedURL() string {
	if p == nil {
		return ""
	}
	return MaskProxyURL(p.URL())
}

func (p *Proxy) FullyMaskedURL() string {
	if p == nil {
		return ""
	}
	return MaskProxyURLCredentials(p.URL())
}

func MaskProxyURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return "<invalid-proxy-url>"
	}
	if u.User != nil {
		username := u.User.Username()
		if username == "" {
			username = "***"
		}
		u.User = url.UserPassword(username, "***")
	}
	return u.String()
}

func MaskProxyURLCredentials(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return "<invalid-proxy-url>"
	}
	if u.User != nil {
		u.User = url.UserPassword("***", "***")
	}
	return u.String()
}

type ProxyWithAccountCount struct {
	Proxy
	AccountCount   int64
	LatencyMs      *int64
	LatencyStatus  string
	LatencyMessage string
	IPAddress      string
	Country        string
	CountryCode    string
	Region         string
	City           string
	QualityStatus  string
	QualityScore   *int
	QualityGrade   string
	QualitySummary string
	QualityChecked *int64
}

type ProxyAccountSummary struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Notes    *string
}
