// Package repository contains repository implementations
package repository

import "context"

var custodyAddresses = map[string]struct{}{
	"0x70997970C51812dc3A010C7d01b50e0d17dc79C8": {},
	"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266": {},
}

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) GetCustodyAddresses(_ context.Context, addresses []string) []string {
	result := make([]string, 0)
	if len(addresses) == 0 {
		for addr := range custodyAddresses {
			result = append(result, addr)
		}
		return result
	}

	for _, address := range addresses {
		if _, ok := custodyAddresses[address]; ok {
			result = append(result, address)
		}
	}
	return result
}
