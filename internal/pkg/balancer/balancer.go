package balancer

import (
	"net/url"
	"sync/atomic"
)

type Balancer struct {
	services []*url.URL
	index    uint32
}

func NewBalancer(services []*url.URL) *Balancer {
	return &Balancer{
		services: services,
		index:    0,
	}
}

func (b *Balancer) Next() *url.URL {
	// Получаем текущий индекс и увеличиваем его атомарно
	i := atomic.AddUint32(&b.index, 1)

	// Возвращаем URL на основе индекса с учетом длины массива
	return b.services[(i-1)%uint32(len(b.services))]
}
