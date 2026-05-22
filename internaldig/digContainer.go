package internaldig

import "go.uber.org/dig"

var DigContainer *dig.Container

func init() {
	DigContainer = dig.New()
}

// Get 通过泛型方式从 Dig 容器中获取实例
// 注意：T 必须是已经在容器中通过 Provide 注册过的类型
func Get[T any]() (T, error) {
	var instance T

	// 创建一个错误变量用于捕获 Invoke 过程中的错误
	var err error

	// 使用 dig.Invoke 注入依赖
	// 我们构造一个匿名函数，其参数为 T，Dig 会尝试解析并填充它
	// 如果 T 是指针类型 (如 *ServiceImpl)，Dig 会注入指针
	// 如果 T 是接口类型 (如 IService)，Dig 会注入接口实现
	err = DigContainer.Invoke(func(val T) {
		instance = val
	})

	if err != nil {
		return instance, err
	}

	return instance, nil
}
func Provide(constructor any) error {
	return DigContainer.Provide(constructor)
}
