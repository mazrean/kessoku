package main

type Config struct {
	Value string
}

func NewConfig() *Config {
	return &Config{Value: "test"}
}

type Service struct {
	config *Config
}

func NewService(config *Config) *Service {
	return &Service{config: config}
}

func main() {
	/*service, err := InitializeService()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Service initialized with config: %+v\n", service.config)*/
}
