package core

import (
	"fmt"
)

// only oriented to llm, should not define client components in the struct, such as tools/skills
type Provider interface {
	// get id of the provider.
	GetID() string
	// get name
	GetName() string
	// auth the model using the given API keys and return the auth result.
	Auth(ModelConfig) *Diagnostic
	// Complete the conversation with messages and tools, return the result.
	Complete(messages []ReActMessage, tools []Tool, modelName string) (Answer, []Diagnostic)
	// stream version of Complete, returns a channel of partial answers
	CompleteStream(messages []ReActMessage, tools []Tool, modelName string) (<-chan Answer, []Diagnostic)
	// get static configuration of the provider.
	GetModelConfig() ModelConfig
	// Init the runtime env for provider if needed.
	Init(ModelConfig) *Diagnostic
}

func ValidateProviders(providers []Provider) []Diagnostic {
	diagnostics := []Diagnostic{}
	hasAvailableModels := false

	for _, provider := range providers {
		modelConfig := provider.GetModelConfig()
		// skip providers with no models configured
		if len(modelConfig.Models) == 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Level:   SeverityWarn,
				Code:    MessageCodeNoAvailableProvider,
				Message: fmt.Sprintf("provider %s has no models configured", provider.GetName()),
			})
			continue
		}

		if err := provider.Auth(modelConfig); err != nil {
			diagnostics = append(diagnostics, *err)
		} else {
			if err := provider.Init(modelConfig); err != nil {
				diagnostics = append(diagnostics, *err)
			} else {
				hasAvailableModels = true
			}
		}
	}

	if !hasAvailableModels {
		diagnostics = append(diagnostics, Diagnostic{
			Level: SeverityError,
			Code:  MessageCodeNoAvailableProvider,
		})
		return diagnostics
	}

	return diagnostics
}

// Abstract ProviderFactory creates a Provider single instance
// rootPath parameter for resolving provider config file paths
type ProviderFactory func(rootPath string) (Provider, *Diagnostic)

// providerFactories registry for independent provider components
var providerFactories = map[string]ProviderFactory{}

// RegisterProviderFactory registers a provider factory by name
func RegisterProviderFactory(name string, factory ProviderFactory) {
	providerFactories[name] = factory
}

// GetProviderFactory returns a registered provider factory by name
func GetProviderFactory(name string) (ProviderFactory, bool) {
	factory, ok := providerFactories[name]
	return factory, ok
}

// GetProviderFactories returns all registered provider factories
func GetProviderFactories() map[string]ProviderFactory {
	return providerFactories
}
