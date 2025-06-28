package main

import "fmt"

const (
	// ExampleArgValue is an example integer value for demonstration
	ExampleArgValue = 10
)

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Interface interface {
	DoSomething() string
}

type ConcreteImpl struct{}

func (c *ConcreteImpl) DoSomething() string {
	return "concrete implementation"
}

func NewConcreteImpl() *ConcreteImpl {
	return &ConcreteImpl{}
}

type Service struct {
	config *Config
	impl   Interface
	value  string
	arg    int
}

func NewService(config *Config, impl Interface, value string, arg int) *Service {
	return &Service{
		config: config,
		impl:   impl,
		value:  value,
		arg:    arg,
	}
}

func main() {
	service := InitializeComplexService(ExampleArgValue)
	fmt.Printf("Service initialized with config: %v, impl: %v, value: %s, arg: %d\n",
		service.config, service.impl.DoSomething(), service.value, service.arg)
}
