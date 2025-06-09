package foundation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/sashabaranov/go-openai"
	"github.com/danielmiessler/fabric/common"
	"github.com/danielmiessler/fabric/core"
	"github.com/danielmiessler/fabric/plugins/db/fsdb"
)

// FabricBridge provides compatibility between Fabric GUI and Fabric core
type FabricBridge struct {
	registry *core.PluginRegistry
	db       *fsdb.Db
}

// NewFabricBridge creates a new compatibility layer
func NewFabricBridge(registry *core.PluginRegistry, db *fsdb.Db) *FabricBridge {
	return &FabricBridge{
		registry: registry,
		db:       db,
	}
}

// LoadPatternContent loads a pattern's content from filesystem
func (fb *FabricBridge) LoadPatternContent(patternID string) (system, user string, tags []string, err error) {
	// Get pattern path
	patternPath := filepath.Join(fb.db.Patterns.Dir, patternID)
	
	// Read system.md
	systemPath := filepath.Join(patternPath, "system.md")
	systemContent, err := os.ReadFile(systemPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to read system.md for pattern '%s': %w", patternID, err)
	}
	system = string(systemContent)
	
	// Try to read user.md (optional)
	userPath := filepath.Join(patternPath, "user.md")
	userContent, err := os.ReadFile(userPath)
	if err == nil {
		user = string(userContent)
	}
	
	// Derive tags (could be enhanced to read from pattern_descriptions.json)
	tags = deriveTagsFromContent(system, patternID)
	
	return system, user, tags, nil
}

// Create compatibility chat options
func CreateChatOptions(
	temperature float64,
	topP float64,
	presencePenalty float64,
	frequencyPenalty float64,
	model string,
) *common.ChatOptions {
	return &common.ChatOptions{
		Temperature:      temperature,
		TopP:             topP,
		PresencePenalty:  presencePenalty,
		FrequencyPenalty: frequencyPenalty,
		Model:            model,
	}
}

// ExecutePatternWithFabric is a compatibility function to execute patterns using Fabric core
func ExecutePatternWithFabric(
	ctx context.Context,
	registry *core.PluginRegistry,
	patternID string,
	input string,
	options *common.ChatOptions,
	stream bool,
	dryRun bool,
) (string, error) {
	// Create a chatter
	chatter, err := registry.GetChatter(
		options.Model,
		options.ModelContextLength, 
		"standard", // default strategy
		stream,
		dryRun,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get chatter: %w", err)
	}
	
	// Create chat request
	chatReq := &common.ChatRequest{
		PatternName: patternID,
		Message: &openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: input,
		},
	}
	
	// Send the request
	session, err := chatter.Send(chatReq, options)
	if err != nil {
		return "", fmt.Errorf("execution failed: %w", err)
	}
	
	// Return the response
	return session.GetLastMessage().Content, nil
}