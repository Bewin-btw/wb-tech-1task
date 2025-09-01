package cache

import (
	"errors"
	"sync"
	"time"

	"wb-tech-1task/internal/models"
)

type Cache struct {
	sync.RWMutex
	data            map[string]*models.Order
	expirationTimes map[string]time.Time
	ttl             time.Duration

	stop chan struct{}
}

func New(ttl time.Duration) *Cache {
	c := &Cache{
		data:            make(map[string]*models.Order),
		expirationTimes: make(map[string]time.Time),
		ttl:             ttl,
		stop:            make(chan struct{}),
	}

	if ttl > 0 {
		go c.startCleanupRoutine()
	}

	return c
}

func (c *Cache) Close() {
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
}

func (c *Cache) Set(order *models.Order) error {
	if order == nil {
		return errors.New("cannot add nil order to cache")
	}
	c.Lock()
	defer c.Unlock()
	orderCopy := c.copyOrder(order)

	c.data[order.OrderUID] = orderCopy

	if c.ttl > 0 {
		c.expirationTimes[order.OrderUID] = time.Now().Add(c.ttl)
	}

	return nil
}

func (c *Cache) Get(orderUID string) (*models.Order, bool, error) {
	c.RLock()
	defer c.RUnlock()
	order, exists := c.data[orderUID]
	if !exists {
		return nil, false, nil
	}

	if c.ttl > 0 {
		if expTime, ok := c.expirationTimes[orderUID]; ok && time.Now().After(expTime) {
			return nil, false, nil
		}
	}

	orderCopy := c.copyOrder(order)
	return orderCopy, true, nil
}

func (c *Cache) GetAll() (map[string]*models.Order, error) {
	c.RLock()
	defer c.RUnlock()

	result := make(map[string]*models.Order, len(c.data))

	now := time.Now()
	for k, v := range c.data {
		if c.ttl > 0 {
			if expTime, ok := c.expirationTimes[k]; ok && now.After(expTime) {
				continue
			}
		}
		result[k] = c.copyOrder(v)
	}

	return result, nil
}

func (c *Cache) Delete(orderUID string) {
	c.Lock()
	defer c.Unlock()

	delete(c.data, orderUID)
	delete(c.expirationTimes, orderUID)
}

func (c *Cache) Cleanup() {
	c.Lock()
	defer c.Unlock()

	now := time.Now()
	for uid, expTime := range c.expirationTimes {
		if now.After(expTime) {
			delete(c.data, uid)
			delete(c.expirationTimes, uid)
		}
	}
}

func (c *Cache) copyOrder(order *models.Order) *models.Order {
	if order == nil {
		return nil
	}
	o := *order
	if len(order.Items) > 0 {
		items := make([]models.Item, len(order.Items))
		copy(items, order.Items)
		o.Items = items
	}
	return &o
}

func (c *Cache) startCleanupRoutine() {
	cleanupInterval := c.ttl / 2
	if cleanupInterval <= 0 {
		cleanupInterval = time.Minute
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Cleanup()
		case <-c.stop:
			return
		}
	}
}

func (c *Cache) Count() int {
	c.RLock()
	defer c.RUnlock()

	if c.ttl == 0 {
		return len(c.data)
	}

	count := 0
	now := time.Now()
	for uid := range c.data {
		if expTime, exists := c.expirationTimes[uid]; exists && now.Before(expTime) {
			count++
		}
	}

	return count
}

func (c *Cache) DBBackup(orders []*models.Order) error {
	c.Lock()
	defer c.Unlock()

	for _, order := range orders {
		if order == nil {
			continue
		}
		orderCopy := c.copyOrder(order)
		c.data[order.OrderUID] = orderCopy

		if c.ttl > 0 {
			c.expirationTimes[order.OrderUID] = time.Now().Add(c.ttl)
		}
	}
	return nil
}
