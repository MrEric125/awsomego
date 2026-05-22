package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// IsolationLevel 事务隔离级别
type IsolationLevel string

const (
	// ReadUncommitted 读未提交
	ReadUncommitted IsolationLevel = "READ UNCOMMITTED"
	// ReadCommitted 读已提交（默认）
	ReadCommitted IsolationLevel = "READ COMMITTED"
	// RepeatableRead 可重复读
	RepeatableRead IsolationLevel = "REPEATABLE READ"
	// Serializable 串行化
	Serializable IsolationLevel = "SERIALIZABLE"
)

// TransactionOptions 事务配置选项
type TransactionOptions struct {
	// Timeout 事务超时时间
	Timeout time.Duration
	// IsolationLevel 隔离级别
	IsolationLevel IsolationLevel
	// RetryCount 重试次数（处理死锁等临时错误）
	RetryCount int
	// RetryDelay 重试间隔
	RetryDelay time.Duration
	// ReadOnly 是否为只读事务（性能优化）
	ReadOnly bool
	// SkipLogging 是否跳过日志记录
	SkipLogging bool
}

// DefaultTransactionOptions 默认事务配置
func DefaultTransactionOptions() TransactionOptions {
	return TransactionOptions{
		Timeout:        30 * time.Second,
		IsolationLevel: ReadCommitted,
		RetryCount:     3,
		RetryDelay:     100 * time.Millisecond,
		ReadOnly:       false,
		SkipLogging:    false,
	}
}

// transactionContextKey 使用自定义类型避免 key 冲突
type transactionContextKey struct{}

// TxContext 事务上下文，包含事务相关信息
type TxContext struct {
	Tx        *gorm.DB
	StartTime time.Time
	Options   TransactionOptions
	Nested    bool // 是否为嵌套事务
}

// GetTxFromContext 从 context 中获取事务（如果存在）
func GetTxFromContext(ctx context.Context) (*gorm.DB, bool) {
	txCtx, ok := ctx.Value(transactionContextKey{}).(*TxContext)
	if !ok || txCtx == nil {
		return nil, false
	}
	return txCtx.Tx, true
}

// GetTxContextFromContext 获取完整的事务上下文
func GetTxContextFromContext(ctx context.Context) (*TxContext, bool) {
	txCtx, ok := ctx.Value(transactionContextKey{}).(*TxContext)
	return txCtx, ok
}

// WithTransaction 执行事务（工业级实现）
func WithTransaction(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error, opts ...TransactionOptions) error {
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

// executeTransaction 执行单次事务
func executeTransaction(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error, options TransactionOptions) error {
	logger := getLogger(ctx)
	startTime := time.Now()

	// 检查是否已有事务（嵌套事务支持）
	if existingTxCtx, ok := GetTxContextFromContext(ctx); ok {
		if !options.SkipLogging {
			logger.Debug("using existing transaction (nested)")
		}
		// 使用现有事务
		ctx = context.WithValue(ctx, transactionContextKey{}, existingTxCtx)
		return fn(ctx)
	}

	// 创建新事务，使用隔离级别配置
	txOpts := &sql.TxOptions{
		ReadOnly: options.ReadOnly,
	}
	
	// 设置事务隔离级别
	if options.IsolationLevel != "" {
		txOpts.Isolation = convertIsolationLevel(options.IsolationLevel)
	}
	
	tx := db.Begin(txOpts)

	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// 设置超时
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// 将事务放入 context
	txCtx := &TxContext{
		Tx:        tx,
		StartTime: startTime,
		Options:   options,
		Nested:    false,
	}
	ctx = context.WithValue(ctx, transactionContextKey{}, txCtx)

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			if !options.SkipLogging {
				logger.Error("transaction panicked",
					zap.Any("panic", p),
					zap.Duration("duration", time.Since(startTime)))
			}
			panic(p)
		}
	}()

	// 执行业务逻辑
	err := fn(ctx)
	if err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			if !options.SkipLogging {
				logger.Error("failed to rollback transaction",
					zap.Error(rbErr),
					zap.Error(err))
			}
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		if !options.SkipLogging {
			logger.Warn("transaction rolled back",
				zap.Error(err),
				zap.Duration("duration", time.Since(startTime)))
		}
		return err
	}

	// 提交事务
	if commitErr := tx.Commit().Error; commitErr != nil {
		if !options.SkipLogging {
			logger.Error("failed to commit transaction",
				zap.Error(commitErr),
				zap.Duration("duration", time.Since(startTime)))
		}
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}

	// 记录成功的事务
	if !options.SkipLogging {
		logger.Info("transaction committed",
			zap.Duration("duration", time.Since(startTime)),
			zap.String("isolation_level", string(options.IsolationLevel)))
	}

	return nil
}

// shouldRetry 判断是否应该重试
func shouldRetry(err error, attempt, maxRetries int) bool {
	if attempt >= maxRetries {
		return false
	}

	errMsg := err.Error()
	// MySQL 死锁错误码: 1213
	// PostgreSQL 死锁错误码: 40P01
	// 临时网络错误等
	retryableErrors := []string{
		"deadlock",
		"Deadlock found",
		"40P01",
		"connection reset",
		"connection refused",
		"i/o timeout",
	}

	for _, retryable := range retryableErrors {
		if contains(errMsg, retryable) {
			return true
		}
	}

	return false
}

// contains 简单的字符串包含检查
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getLogger 从 context 或全局获取 logger
func getLogger(ctx context.Context) *zap.Logger {
	// 这里可以集成你的日志系统
	// 暂时返回默认 logger
	logger, _ := zap.NewProduction()
	return logger
}

// SetIsolationLevel 设置事务隔离级别（可选）
func SetIsolationLevel(db *gorm.DB, level IsolationLevel) *gorm.DB {
	return db.Exec(fmt.Sprintf("SET TRANSACTION ISOLATION LEVEL %s", level))
}

// convertIsolationLevel 将自定义隔离级别转换为 sql.IsolationLevel
func convertIsolationLevel(level IsolationLevel) sql.IsolationLevel {
	switch level {
	case ReadUncommitted:
		return sql.LevelReadUncommitted
	case ReadCommitted:
		return sql.LevelReadCommitted
	case RepeatableRead:
		return sql.LevelRepeatableRead
	case Serializable:
		return sql.LevelSerializable
	default:
		return sql.LevelDefault
	}
}
