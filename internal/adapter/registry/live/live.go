// SPDX-License-Identifier: AGPL-3.0-only

// Package live builds the registries a check consults online: RDAP,
// SIRENE and BODACC, each through an HTTP client allowed its own hosts
// only.
package live

import (
	"github.com/arnakoff/api/internal/adapter/registry/bodacc"
	"github.com/arnakoff/api/internal/adapter/registry/rdap"
	"github.com/arnakoff/api/internal/adapter/registry/sirene"
	"github.com/arnakoff/api/internal/app"
	"github.com/arnakoff/api/internal/domain"
	"github.com/arnakoff/api/internal/platform/httpclient"
)

// Registries returns the RDAP, SIRENE and BODACC clients of one of
// instances processes running at once, each keeping to its share of the
// registries' limits per IP address; the lists are left to the caller.
// Nothing is fetched until a check asks.
func Registries(instances int) (app.Registries, error) {
	domains, err := domainRegistry()
	if err != nil {
		return app.Registries{}, err
	}
	sc, err := httpclient.New(httpclient.Config{AllowedHosts: sirene.Hosts()})
	if err != nil {
		return app.Registries{}, err
	}
	companies, err := sirene.New(sc, sirene.BaseURL, instances)
	if err != nil {
		return app.Registries{}, err
	}
	bc, err := httpclient.New(httpclient.Config{AllowedHosts: bodacc.Hosts()})
	if err != nil {
		return app.Registries{}, err
	}
	notices, err := bodacc.New(bc, bodacc.BaseURL)
	if err != nil {
		return app.Registries{}, err
	}
	return app.Registries{Domains: domains, Companies: companies, Notices: notices}, nil
}

// domainRegistry returns the RDAP client: the IANA bootstrap through one
// HTTP client, each registry's server through a client allowed its hosts.
func domainRegistry() (*rdap.Client, error) {
	iana, err := httpclient.New(httpclient.Config{AllowedHosts: rdap.BootstrapHosts()})
	if err != nil {
		return nil, err
	}
	return rdap.New(iana, rdap.BootstrapURL, func(hosts []string) (rdap.Getter, error) {
		c, err := httpclient.New(httpclient.Config{AllowedHosts: hosts})
		if err != nil {
			return nil, err
		}
		return c, nil
	})
}

// Infos describes the registries Registries returns, in the same order,
// for app.NewSourceList.
func Infos() []domain.SourceInfo {
	return []domain.SourceInfo{rdap.Info(), sirene.Info(), bodacc.Info()}
}
