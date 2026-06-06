// Package domain contains internal data models
package domain

import "context"

type Repository interface {
	GetCustodyAddresses(ctx context.Context, addresses []string) []string
}
