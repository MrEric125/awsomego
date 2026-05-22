# 工业级事务管理最佳实践

## 📋 目录

- [架构设计](#架构设计)
- [核心特性](#核心特性)
- [使用指南](#使用指南)
- [最佳实践](#最佳实践)
- [性能优化](#性能优化)
- [常见问题](#常见问题)

---

## 🏗️ 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────┐
│                  Service Layer                   │
│  (定义事务边界，配置事务选项)                      │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│            Transaction Manager                   │
│  - 事务生命周期管理                               │
│  - 重试机制                                      │
│  - 超时控制                                      │
│  - 监控日志                                      │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│             Repository Layer                     │
│  - 从 Context 获取事务                           │
│  - 统一使用 BaseRepository                       │
│  - 自动事务传播                                  │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│              Database (GORM)                     │
└─────────────────────────────────────────────────┘
```

### 关键组件

1. **Transaction Manager** (`transaction_v2.go`)
   - 统一的事务入口
   - 自动重试、超时、监控
   - 嵌套事务支持

2. **BaseRepository** (`base_repository.go`)
   - 所有 Repository 的基类
   - 自动从事务上下文获取连接
   - 提供通用 CRUD 方法

3. **Context 传递**
   - 通过 `context.Context` 传递事务
   - 无需显式传递 `*gorm.DB`
   - 支持事务自动传播

---

## ✨ 核心特性

### 1. 自动重试机制

```go
opts := database.TransactionOptions{
    RetryCount: 3,              // 最多重试3次
    RetryDelay: 100 * time.Millisecond,
}

database.WithTransaction(ctx, db, func(ctx context.Context) error {
    // 死锁等临时错误会自动重试
    return businessLogic(ctx)
}, opts)
```

**自动重试的场景：**
- 数据库死锁 (Deadlock)
- 临时网络错误
- 连接超时
- 资源竞争

### 2. 超时控制

```go
opts := database.TransactionOptions{
    Timeout: 10 * time.Second,  // 10秒超时
}
```

防止长时间运行的事务占用数据库资源。

### 3. 隔离级别控制

```go
// 读未提交（最低隔离）
database.ReadUncommitted

// 读已提交（默认，推荐）
database.ReadCommitted

// 可重复读
database.RepeatableRead

// 串行化（最高隔离，金融场景）
database.Serializable
```

### 4. 嵌套事务支持

```go
// 外层事务
database.WithTransaction(ctx, db, func(outerCtx context.Context) error {
    createUser(outerCtx)
    
    // 内层事务：自动复用外层事务
    database.WithTransaction(outerCtx, db, func(innerCtx context.Context) error {
        createOrder(innerCtx)
        return nil
    })
    
    return nil
})
```

### 5. 只读事务优化

```go
opts := database.TransactionOptions{
    ReadOnly: true,  // 数据库可以优化性能
}

database.WithTransaction(ctx, db, func(ctx context.Context) error {
    return queryData(ctx)
}, opts)
```

### 6. 监控和日志

每个事务自动记录：
- 执行时长
- 是否成功/失败
- 重试次数
- 隔离级别

---

## 📖 使用指南

### 基础用法

```go
import "awesome/internal/database"

func CreateUser(name, email string) error {
    ctx := context.Background()
    
    err := database.WithTransaction(ctx, db, func(ctx context.Context) error {
        user := &models.User{
            Name:  name,
            Email: email,
        }
        return userRepo.Create(ctx, user)
    })
    
    return err
}
```

### 自定义配置

```go
opts := database.TransactionOptions{
    Timeout:        5 * time.Second,
    IsolationLevel: database.ReadCommitted,
    RetryCount:     3,
    RetryDelay:     100 * time.Millisecond,
    ReadOnly:       false,
    SkipLogging:    false,
}

err := database.WithTransaction(ctx, db, businessLogic, opts)
```

### Repository 实现

```go
type UserRepositoryImpl struct {
    *repository.BaseRepository  // 继承基类
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *models.User) error {
    // 自动使用事务（如果存在）
    return r.BaseRepository.Create(ctx, user)
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id int) (*models.User, error) {
    var user models.User
    // 自动使用事务（如果存在）
    if err := r.FindByID(ctx, &user, id); err != nil {
        return nil, err
    }
    return &user, nil
}
```

---

## 🎯 最佳实践

### 1. 事务边界设计

✅ **推荐：**
```go
// Service 层定义事务边界
func (s *UserService) CreateUserWithOrder(...) error {
    return database.WithTransaction(ctx, db, func(ctx context.Context) error {
        s.userRepo.Create(ctx, user)
        s.orderRepo.Create(ctx, order)
        return nil
    })
}
```

❌ **避免：**
```go
// 不要在 Repository 层开启事务
func (r *UserRepository) Create(user *User) error {
    tx := db.Begin()  // ❌ 错误做法
    // ...
}
```

### 2. 选择合适的隔离级别

| 场景 | 隔离级别 | 说明 |
|------|---------|------|
| 普通业务 | `ReadCommitted` | 默认推荐，平衡性能和一致性 |
| 报表查询 | `ReadUncommitted` | 允许脏读，性能最高 |
| 财务统计 | `RepeatableRead` | 确保多次读取一致 |
| 转账交易 | `Serializable` | 最高一致性，性能较低 |

### 3. 合理设置超时时间

```go
// 简单操作：短超时
opts := TransactionOptions{Timeout: 5 * time.Second}

// 复杂业务：中等超时
opts := TransactionOptions{Timeout: 15 * time.Second}

// 批量操作：长超时
opts := TransactionOptions{Timeout: 60 * time.Second}
```

### 4. 避免长时间持有事务

✅ **推荐：**
```go
database.WithTransaction(ctx, db, func(ctx context.Context) error {
    // 只做数据库操作
    createUser(ctx)
    createOrder(ctx)
    return nil
})
```

❌ **避免：**
```go
database.WithTransaction(ctx, db, func(ctx context.Context) error {
    createUser(ctx)
    
    // ❌ 不要在外部 API 调用时持有事务
    callExternalAPI()  // 可能导致事务长时间锁定
    
    createOrder(ctx)
    return nil
})
```

### 5. 使用原子操作处理并发

```go
// ✅ 推荐：原子操作减少库存
func DecreaseStock(ctx context.Context, productID, quantity int) error {
    result := db.Model(&Product{}).
        Where("id = ? AND stock >= ?", productID, quantity).
        UpdateColumn("stock", gorm.Expr("stock - ?", quantity))
    
    if result.RowsAffected == 0 {
        return errors.New("库存不足")
    }
    return nil
}

// ❌ 避免：先查后改（有竞态条件）
product := GetProduct(productID)
if product.Stock < quantity {
    return errors.New("库存不足")
}
product.Stock -= quantity
Save(product)
```

### 6. 错误处理

```go
err := database.WithTransaction(ctx, db, func(ctx context.Context) error {
    if err := step1(ctx); err != nil {
        return fmt.Errorf("step 1 failed: %w", err)  // 包装错误
    }
    
    if err := step2(ctx); err != nil {
        return fmt.Errorf("step 2 failed: %w", err)
    }
    
    return nil
})

if err != nil {
    // 事务已自动回滚
    log.Error("Transaction failed", zap.Error(err))
    return err
}
```

### 7. 批量操作优化

```go
// ✅ 推荐：批量插入
func BatchCreateUsers(ctx context.Context, users []User) error {
    return database.WithTransaction(ctx, db, func(ctx context.Context) error {
        // 分批插入，每批100条
        for i := 0; i < len(users); i += 100 {
            end := i + 100
            if end > len(users) {
                end = len(users)
            }
            batch := users[i:end]
            
            if err := db.WithContext(ctx).CreateInBatches(batch, 100).Error; err != nil {
                return err
            }
        }
        return nil
    }, TransactionOptions{Timeout: 60 * time.Second})
}
```

---

## ⚡ 性能优化

### 1. 使用只读事务

```go
// 查询操作使用只读事务，数据库可以优化
opts := TransactionOptions{ReadOnly: true}
database.WithTransaction(ctx, db, queryFunc, opts)
```

### 2. 减少事务范围

```go
// ❌ 避免：大范围事务
database.WithTransaction(ctx, db, func(ctx context.Context) error {
    data := queryLargeDataset(ctx)  // 耗时查询
    processData(data)               // CPU 密集计算
    saveResult(ctx)                 // 保存结果
    return nil
})

// ✅ 推荐：小范围事务
data := queryLargeDataset(ctx)      // 事务外查询
result := processData(data)         // 事务外计算

database.WithTransaction(ctx, db, func(ctx context.Context) error {
    return saveResult(ctx, result)  // 只包裹必要的写操作
})
```

### 3. 合理使用索引

确保事务中的查询使用索引，避免全表扫描导致长时间锁表。

### 4. 批量操作

```go
// ✅ 批量插入优于逐条插入
db.CreateInBatches(users, 100)

// ❌ 避免循环单条插入
for _, user := range users {
    db.Create(&user)  // 慢！
}
```

---

## ❓ 常见问题

### Q1: 为什么我的事务没有回滚？

**A:** 检查以下几点：
1. Repository 是否使用了 `BaseRepository`？
2. 是否正确传递了 `context.Context`？
3. 是否所有操作都在同一个事务中？

```go
// ✅ 正确：传递 ctx
userRepo.Create(ctx, user)

// ❌ 错误：没有传递 ctx
userRepo.Create(context.Background(), user)
```

### Q2: 如何处理死锁？

**A:** 配置自动重试：
```go
opts := TransactionOptions{
    RetryCount: 3,
    RetryDelay: 100 * time.Millisecond,
}
```

同时优化：
- 保持事务简短
- 按相同顺序访问资源
- 使用合适的隔离级别

### Q3: 嵌套事务如何工作？

**A:** 内层事务会复用外层事务，不会创建新事务：
```go
// 外层事务
WithTransaction(ctx, db, func(outerCtx context.Context) error {
    // 内层：使用同一个事务
    WithTransaction(outerCtx, db, func(innerCtx context.Context) error {
        // ...
    })
})
```

### Q4: 如何监控事务性能？

**A:** 事务自动记录日志，包括：
- 执行时长
- 是否成功
- 重试次数

可以集成监控系统（如 Prometheus）收集指标。

### Q5: 什么时候不使用事务？

**A:** 以下场景不需要事务：
- 单一查询操作
- 不涉及数据一致性的操作
- 只读操作（除非需要一致性快照）

---

## 📊 对比：改造前后

### 改造前（有问题）

```go
// Service 层
err := database.ExecuteInTransaction(db, func(tx *gorm.DB) error {
    userRepo.Create(ctx, user)  // ❌ 不使用 tx
    orderRepo.Create(ctx, order) // ❌ 不使用 tx
    return nil
})
// 问题：Repository 使用独立连接，事务无效
```

### 改造后（工业级）

```go
// Service 层
opts := TransactionOptions{
    Timeout: 10 * time.Second,
    RetryCount: 3,
}
err := database.WithTransaction(ctx, db, func(ctx context.Context) error {
    userRepo.Create(ctx, user)  // ✅ 自动使用事务
    orderRepo.Create(ctx, order) // ✅ 自动使用事务
    return nil
}, opts)

// Repository 层
type UserRepositoryImpl struct {
    *BaseRepository  // ✅ 继承基类
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *User) error {
    return r.BaseRepository.Create(ctx, user)  // ✅ 自动获取事务
}
```

---

## 🔧 迁移指南

### 步骤 1: 更新 Repository

```go
// 旧代码
type UserRepositoryImpl struct {
    db *gorm.DB
}

// 新代码
type UserRepositoryImpl struct {
    *BaseRepository
}

func NewUserRepository(db *gorm.DB) *UserRepositoryImpl {
    return &UserRepositoryImpl{
        BaseRepository: NewBaseRepository(db),
    }
}
```

### 步骤 2: 简化 Repository 方法

```go
// 旧代码
func (r *UserRepositoryImpl) Create(ctx context.Context, user *User) error {
    db := r.db
    if tx, ok := database.GetTxFromContext(ctx); ok {
        db = tx
    }
    return db.Create(user).Error
}

// 新代码
func (r *UserRepositoryImpl) Create(ctx context.Context, user *User) error {
    return r.BaseRepository.Create(ctx, user)
}
```

### 步骤 3: 更新 Service 层

```go
// 旧代码
tm := database.NewTransactionManager()
err := tm.WithTransaction(ctx, db, func(ctx context.Context, tx *gorm.DB) error {
    // ...
})

// 新代码
opts := database.TransactionOptions{
    Timeout: 10 * time.Second,
    RetryCount: 3,
}
err := database.WithTransaction(ctx, db, func(ctx context.Context) error {
    // ...
}, opts)
```

---

## 📚 参考资料

- [GORM Transactions](https://gorm.io/docs/transactions.html)
- [MySQL Transaction Isolation](https://dev.mysql.com/doc/refman/8.0/en/innodb-transaction-isolation-levels.html)
- [Database Deadlocks](https://en.wikipedia.org/wiki/Deadlock)

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进事务管理系统！
