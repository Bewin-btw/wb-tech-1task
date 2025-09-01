package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"wb-tech-1task/internal/models"
)

func TestCache_SetGet(t *testing.T) {
	cache := New(10 * time.Minute)
	defer cache.Close()

	order := &models.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK123",
	}

	err := cache.Set(order)
	assert.NoError(t, err)

	retrieved, exists, err := cache.Get("test123")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, order.OrderUID, retrieved.OrderUID)
}

func TestCache_Expiration(t *testing.T) {
	cache := New(100 * time.Millisecond)
	defer cache.Close()

	order := &models.Order{OrderUID: "test123"}
	cache.Set(order)

	time.Sleep(150 * time.Millisecond)

	_, exists, _ := cache.Get("test123")
	assert.False(t, exists)
}

func TestCache_CopyIsolation(t *testing.T) {
	cache := New(10 * time.Minute)
	defer cache.Close()

	original := &models.Order{
		OrderUID:    "test123",
		TrackNumber: "ORIGINAL",
	}

	cache.Set(original)
	retrieved, _, _ := cache.Get("test123")

	retrieved.TrackNumber = "MODIFIED"

	retrievedAgain, _, _ := cache.Get("test123")
	assert.Equal(t, "ORIGINAL", retrievedAgain.TrackNumber)
}
