package foundation

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExecutionManager handles pattern execution
type ExecutionManager struct {
	app           *FabricApp
	config        *FabricConfig
	activeRequest context.CancelFunc // For canceling ongoing requests
}

// NewExecutionManager creates a new execution manager
func NewExecutionManager(app *FabricApp, config *FabricConfig) *ExecutionManager {
	return &ExecutionManager{
		app:    app,
		config: config,
	}
}

// ExecutePattern runs a pattern with the given configuration
func (em *ExecutionManager) ExecutePattern(config ExecutionConfig) (*ExecutionResult, error) {
	startTime := time.Now()
	
	// Cancel any existing execution
	if em.activeRequest != nil {
		em.activeRequest()
	}
	
	// Create a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	em.activeRequest = cancel
	
	// Prepare to collect execution result
	result := &ExecutionResult{
		PatternID:  config.PatternID,
		Timestamp:  startTime,
		Success:    false,
	}
	
	// Find the pattern
	var pattern Pattern
	found := false
	for _, p := range em.app.state.LoadedPatterns {
		if p.ID == config.PatternID {
			pattern = p
			found = true
			break
		}
	}
	
	if !found {
		err := fmt.Errorf("pattern not found: %s", config.PatternID)
		result.Error = err
		return result, err
	}
	
	log.Printf("Executing pattern %s with model %s (%s)", pattern.ID, config.Model, config.Vendor)
	
	// Create compatible chat options using the helper function
	chatOptions := CreateChatOptions(
		config.Temperature,
		config.TopP,
		config.PresencePenalty,
		config.FrequencyPenalty,
		config.Model,
	)
	
	// Execute the pattern using the compatibility function
	execFunc := func() (string, error) {
		response, err := ExecutePatternWithFabric(
			ctx,
			em.config.registry,
			pattern.ID,
			config.Input,
			chatOptions,
			config.Stream,
			config.DryRun,
		)
		if err != nil {
			return "", fmt.Errorf("execution failed: %w", err)
		}
		return response, nil
	}
	
	// Execute with timeout handling
	resultChan := make(chan string)
	errChan := make(chan error)
	
	go func() {
		output, err := execFunc()
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- output
	}()
	
	// Wait for result, error, or timeout
	select {
	case <-ctx.Done():
		err := fmt.Errorf("execution timed out after %v", time.Since(startTime))
		result.Error = err
		return result, err
		
	case err := <-errChan:
		result.Error = err
		result.ExecutionTime = time.Since(startTime)
		return result, err
		
	case output := <-resultChan:
		result.Output = output
		result.Success = true
		result.ExecutionTime = time.Since(startTime)
		// We don't have token count information from the API directly
		// Estimate based on text length (very rough approximation)
		result.TokensUsed = estimateTokenCount(config.Input) + estimateTokenCount(output)
		return result, nil
	}
}

// CancelExecution cancels the active execution if any
func (em *ExecutionManager) CancelExecution() {
	if em.activeRequest != nil {
		em.activeRequest()
		em.activeRequest = nil
	}
}

// Helper functions

// estimateTokenCount provides a rough estimate of token count based on text length
// This is not accurate but gives a rough idea. A proper implementation would use
// the tokenizer from the specific model being used.
func estimateTokenCount(text string) int {
	// Rough approximation: 1 token is about 4 characters for English text
	return len(text) / 4
}

// ExecutePatternWithStreamHandler executes a pattern with streaming response
func (em *ExecutionManager) ExecutePatternWithStreamHandler(
	config ExecutionConfig,
	onChunk func(chunk string),
	onComplete func(result *ExecutionResult),
	onError func(err error),
) {
	startTime := time.Now()
	
	// Cancel any existing execution
	if em.activeRequest != nil {
		em.activeRequest()
	}
	
	// Create a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	em.activeRequest = cancel
	
	// Find the pattern
	var pattern Pattern
	found := false
	for _, p := range em.app.state.LoadedPatterns {
		if p.ID == config.PatternID {
			pattern = p
			found = true
			break
		}
	}
	
	if !found {
		err := fmt.Errorf("pattern not found: %s", config.PatternID)
		onError(err)
		return
	}
	
	// Set streaming to true for this execution
	config.Stream = true
	
	log.Printf("Streaming pattern %s with model %s (%s)", pattern.ID, config.Model, config.Vendor)
	
	// Create compatible chat options with streaming enabled
	chatOptions := CreateChatOptions(
		config.Temperature,
		config.TopP,
		config.PresencePenalty,
		config.FrequencyPenalty,
		config.Model,
	)
	
	go func() {
		// Execute with the compatibility function
		output, err := ExecutePatternWithFabric(
			ctx,
			em.config.registry,
			pattern.ID,
			config.Input,
			chatOptions,
			true, // stream
			config.DryRun,
		)
		
		if err != nil {
			onError(fmt.Errorf("execution failed: %w", err))
			return
		}
		
		// For now, just send the full response as one chunk
		// In a real implementation, we would use a proper streaming API
		onChunk(output)
		
		// All chunks received, build the final result
		result := &ExecutionResult{
			PatternID:     config.PatternID,
			Output:        output,
			Success:       true,
			Timestamp:     startTime,
			ExecutionTime: time.Since(startTime),
			TokensUsed:    estimateTokenCount(config.Input) + estimateTokenCount(output),
		}
		
		onComplete(result)
	}()
}