// Package repository contains repository implementations
package repository

import "context"

var custodyAddresses = map[string]struct{}{
	"0x0578E5EA652C62DB20F4475F685A4b587314A30f": {},
	"0xA3E36262f6899e27bB4B1802e8298e843E74CBC7": {},
}

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) GetCustodyAddresses(_ context.Context, addresses []string) []string {
	result := make([]string, 0)
	for _, address := range addresses {
		if _, ok := custodyAddresses[address]; ok {
			result = append(result, address)
		}
	}
	return result
}
