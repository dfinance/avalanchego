// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metercacher

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/utils/timer"
)

var _ cache.Cacher = &Cache{}

type Cache struct {
	metrics
	cache.Cacher

	clock timer.Clock
}

func New(
	namespace string,
	registerer prometheus.Registerer,
	cache cache.Cacher,
) (cache.Cacher, error) {
	meterCache := &Cache{Cacher: cache}
	return meterCache, meterCache.metrics.Initialize(namespace, registerer)
}

func (c *Cache) Put(key, value interface{}) {
	start := c.clock.Time()
	c.Cacher.Put(key, value)
	end := c.clock.Time()
	c.put.Observe(float64(end.Sub(start)))
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	start := c.clock.Time()
	value, has := c.Cacher.Get(key)
	end := c.clock.Time()
	c.get.Observe(float64(end.Sub(start)))
	if has {
		c.hit.Inc()
	} else {
		c.miss.Inc()
	}

	return value, has
}
