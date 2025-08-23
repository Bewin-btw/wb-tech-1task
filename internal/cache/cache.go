package cache

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
	"wb-tech-1task/internal/models"
)

// ttl, expirationTimes для возможности добавления кеша с ттл, LRU можно добавть в теории для прода
type Cache struct {
	sync.RWMutex
	data            map[string]*models.Order
	expirationTimes map[string]time.Time
	ttl             time.Duration
}

func New(ttl time.Duration) *Cache {
	c := &Cache{
		data:            make(map[string]*models.Order),
		expirationTimes: make(map[string]time.Time),
		ttl:             ttl,
	}

	if ttl > 0 {
		go c.startCleanupRoutine()
	}

	return c
}

func (c *Cache) Set(order *models.Order) error {
	if order == nil {
		return errors.New("cannot add nil order to cache")
	}
	c.Lock()
	defer c.Unlock()
	orderCopy, err := c.deepCopyOrder(order)
	if err != nil {
		return err
	}

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
		if expTime, exists := c.expirationTimes[order.OrderUID]; exists && time.Now().After(expTime) {
			return nil, false, nil
		}
	}

	orderCopy, err := c.deepCopyOrder(order)
	if err != nil {
		return nil, false, err
	}

	return orderCopy, true, nil
}

func (c *Cache) GetAll() (map[string]*models.Order, error) {
	c.RLock()
	defer c.RUnlock()

	result := make(map[string]*models.Order, len(c.data))

	for k, v := range c.data {
		if c.ttl > 0 {
			if expTime, exists := c.expirationTimes[k]; exists && time.Now().After(expTime) {
				continue
			}
		}
		orderCopy, err := c.deepCopyOrder(v)
		if err != nil {
			return nil, err
		}
		result[k] = orderCopy
	}

	return result, nil
}

func (c *Cache) Delete(orderUID string) {
	c.Lock()
	defer c.Unlock()

	delete(c.data, orderUID)
	delete(c.expirationTimes, orderUID)
}

// для удаления просроченых записей кеша
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

func (c *Cache) deepCopyOrder(order *models.Order) (*models.Order, error) {
	data, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}

	var orderCopy models.Order

	err = json.Unmarshal(data, &orderCopy)
	if err != nil {
		return nil, err
	}

	return &orderCopy, nil
}

func (c *Cache) startCleanupRoutine() {
	cleanupInterval := c.ttl / 2
	if cleanupInterval <= 0 {
		cleanupInterval = time.Minute
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.Cleanup()
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
		orderCopy, err := c.deepCopyOrder(order)
		if err != nil {
			return err
		}
		c.data[order.OrderUID] = orderCopy

		if c.ttl > 0 {
			c.expirationTimes[order.OrderUID] = time.Now().Add(c.ttl)
		}
	}
	return nil
}
