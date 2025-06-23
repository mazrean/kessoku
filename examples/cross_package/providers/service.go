package providers

// ExternalService represents a service from another package.
type ExternalService struct {
	Config *ExternalConfig
	Name   string
}

// NewExternalService creates a new external service.
func NewExternalService(config *ExternalConfig) *ExternalService {
	return &ExternalService{
		Config: config,
		Name:   "External Service",
	}
}

// GetInfo returns service information.
func (s *ExternalService) GetInfo() string {
	return s.Name + " with DB: " + s.Config.DatabaseURL
}
