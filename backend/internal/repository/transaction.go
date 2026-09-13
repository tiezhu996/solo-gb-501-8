package repository

import (
	"context"

	"gorm.io/gorm"
)

type transactionContextKey struct{}

// Transactor keeps all repository calls made by fn on the same database
// transaction. The transaction is carried in context so services do not need
// to leak *gorm.DB through every repository interface.
type Transactor interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}

type transactor struct{ db *gorm.DB }

func NewTransactor(db *gorm.DB) Transactor { return &transactor{db: db} }

func (t *transactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, transactionContextKey{}, tx))
	})
}

func dbForContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(transactionContextKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}
