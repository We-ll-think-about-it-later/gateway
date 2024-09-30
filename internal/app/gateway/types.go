package gateway

import (
	"fmt"
	"gateway/internal/pkg/balancer"
	"log"
	"net/url"
	"strings"
)

// newUpstreamServices создает UpstreamServices с балансировщиком для сервисов аутентификации.
func newUpstreamServices(serviceAddresses string) (*UpstreamServices, error) {
	urls, err := parseUpstreamURLs(serviceAddresses)
	if err != nil {
		return nil, err
	}

	balancer := balancer.NewBalancer(urls)
	return &UpstreamServices{IdentityService: balancer}, nil
}

// parseUpstreamURLs разбирает строки URL в список *url.URL.
func parseUpstreamURLs(addresses string) ([]*url.URL, error) {
	services := strings.Split(addresses, ",")
	urls := make([]*url.URL, 0, len(services))

	for _, service := range services {
		service = strings.TrimSpace(service)
		u, err := url.Parse(service)
		if err != nil {
			log.Printf("Неверный URL для сервиса %s: %v", service, err)
			continue
		}
		urls = append(urls, u)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("не предоставлены действительные адреса upstream сервисов")
	}

	return urls, nil
}
