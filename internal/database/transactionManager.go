package database

import (
	"context"

	"gorm.io/gorm"

	"fmt"
	"time"

	"go.uber.org/zap"
)

type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) TransactionManager {
	return TransactionManager{
		db: db,
	}
}

// WithTransaction 执行事务（工业级实现）
func (tm *TransactionManager) WithTransaction(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error, opts ...TransactionOptions) error {
	options := DefaultTransactionOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	logger := getLogger(ctx)

	var lastErr error
	for attempt := 0; attempt <= options.RetryCount; attempt++ {
		if attempt > 0 {
			if !options.SkipLogging {
				logger.Warn("retrying transaction",
					zap.Int("attempt", attempt+1),
					zap.Int("max_retries", options.RetryCount+1),
					zap.Error(lastErr))
			}
			time.Sleep(options.RetryDelay)
		}

		err := executeTransaction(ctx, db, fn, options)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否应该重试（死锁、超时等临时错误）
		if !shouldRetry(err, attempt, options.RetryCount) {
			break
		}
	}

	return fmt.Errorf("transaction failed after %d attempts: %w", options.RetryCount+1, lastErr)
}
