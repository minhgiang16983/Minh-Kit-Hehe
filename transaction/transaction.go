package transaction

import (
	"context"
	"errors"

	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gorm.io/gorm"
)

type TransactionInterface interface {
	BeginTransaction(ctx context.Context) (context.Context, CancelTransactionFunc)
	GetTx(ctx context.Context) (*gorm.DB, bool)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Transaction struct {
	db *gorm.DB
	tr trace.Tracer
}

type CancelTransactionFunc func(err error) error

type transactionKey struct{}

func New(db *gorm.DB) TransactionInterface {
	var tr trace.Tracer = nil
	if tracing.GetGlobalTracingConfig().IsEnableTracing() {
		name := tracing.GetGlobalTracingConfig().ServiceName
		tr = otel.GetTracerProvider().Tracer(name)
	}
	return &Transaction{
		db: db,
		tr: tr,
	}
}

func (t *Transaction) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, func()) {
	if t.tr == nil {
		return ctx, func() {}
	}
	ctx2, span := t.tr.Start(ctx, name)
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx2, func() { span.End() }
}

func (t *Transaction) BeginTransaction(ctx context.Context) (context.Context, CancelTransactionFunc) {
	ctx, end := t.StartSpan(ctx, "BeginTransaction")
	defer end()
	tx := t.db.WithContext(ctx).Begin()

	newCtx := context.WithValue(ctx, transactionKey{}, tx)
	return newCtx, func(err error) error {
		if r := recover(); r != nil || err != nil {
			t.Rollback(newCtx)
			if err != nil {
				return err
			}

			return errors.New("occurred panic")
		}

		if err = t.Commit(newCtx); err != nil {
			t.Rollback(newCtx)
			return err
		}

		return nil
	}

}

func (t *Transaction) GetTx(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(transactionKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return nil, false
	}

	return tx, true
}

func (t *Transaction) Commit(ctx context.Context) error {
	ctx, end := t.StartSpan(ctx, "Commit")
	defer end()

	tx, ok := t.GetTx(ctx)
	if !ok {
		return errors.New("transaction not found")
	}

	// Check error context trc khi commit
	if ctx.Err() != nil {
		tx.Rollback()
		return ctx.Err()
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return errors.New("failed to commit transaction")
	}
	return nil
}

func (t *Transaction) Rollback(ctx context.Context) error {
	ctx, end := t.StartSpan(ctx, "Rollback")
	defer end()
	tx, ok := t.GetTx(ctx)
	if !ok {
		return errors.New("transaction not found")
	}

	if err := tx.Rollback().Error; err != nil {
		return errors.New("failed to rollback transaction")
	}
	return nil
}

func GetTxFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(transactionKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return nil, false
	}
	return tx, true
}

func WithTransaction(ctx context.Context, db *gorm.DB) (context.Context, *gorm.DB) {
	tx := db.Begin()
	newCtx := context.WithValue(ctx, transactionKey{}, tx)
	return newCtx, tx
}

func GetTxFromContextOrDefault(ctx context.Context, db *gorm.DB) *gorm.DB {
	tx, ok := GetTxFromContext(ctx)
	if !ok {
		return db
	}
	return tx
}
