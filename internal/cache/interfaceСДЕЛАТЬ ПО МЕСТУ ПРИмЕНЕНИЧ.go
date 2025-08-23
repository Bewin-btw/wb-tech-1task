package cache

import "wb-tech-1task/internal/models"

type OrderCache interface {
	Set(order *models.Order) error
	Get(orderUID string) (*models.Order, bool, error)
	GetAll() (map[string]*models.Order, error)
	Delete(orderUID string)
	Count() int
}
