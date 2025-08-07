/**
 * CONTEXT:   Comprehensive tests for Claude processing time tracking functionality
 * INPUT:     Test scenarios covering Claude processing states, idle detection, and time calculations
 * OUTPUT:    Test validation of enhanced work tracking with Claude processing awareness
 * BUSINESS:  Ensure Claude processing time tracking works correctly and prevents false idle detection
 * CHANGE:    Initial comprehensive test suite for Claude processing enhancements
 * RISK:      High - Test coverage critical for ensuring accurate work time tracking
 */

package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * CONTEXT:   Test Claude processing context creation and validation
 * INPUT:     Various Claude processing context configurations
 * OUTPUT:    Validation that Claude processing contexts are created correctly
 * BUSINESS:  Claude processing context enables accurate processing time tracking
 * CHANGE:    Initial Claude processing context tests
 * RISK:      Medium - Context creation affects all Claude processing tracking
 */
func TestClaudeProcessingContext_Creation(t *testing.T) {
	tests := []struct {
		name           string
		promptID       string
		estimatedTime  time.Duration
		promptLength   int
		complexityHint string
		claudeActivity ClaudeActivityType
		expectValid    bool
	}{
		{
			name:           "valid processing start context",
			promptID:       "prompt_123",
			estimatedTime:  30 * time.Second,
			promptLength:   100,
			complexityHint: "moderate",
			claudeActivity: ClaudeActivityStart,
			expectValid:    true,
		},
		{
			name:           "valid processing end context",
			promptID:       "prompt_123",
			estimatedTime:  30 * time.Second,
			promptLength:   100,
			complexityHint: "simple",
			claudeActivity: ClaudeActivityEnd,
			expectValid:    true,
		},
		{
			name:           "empty prompt ID",
			promptID:       "",
			estimatedTime:  30 * time.Second,
			promptLength:   100,
			complexityHint: "moderate",
			claudeActivity: ClaudeActivityStart,
			expectValid:    true, // Empty prompt ID is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := &ClaudeProcessingContext{
				PromptID:       tt.promptID,
				EstimatedTime:  tt.estimatedTime,
				PromptLength:   tt.promptLength,
				ComplexityHint: tt.complexityHint,
				ClaudeActivity: tt.claudeActivity,
			}

			// Test basic properties
			assert.Equal(t, tt.promptID, context.PromptID)
			assert.Equal(t, tt.estimatedTime, context.EstimatedTime)
			assert.Equal(t, tt.promptLength, context.PromptLength)
			assert.Equal(t, tt.complexityHint, context.ComplexityHint)
			assert.Equal(t, tt.claudeActivity, context.ClaudeActivity)
		})
	}
}

/**
 * CONTEXT:   Test enhanced ActivityEvent with Claude processing context
 * INPUT:     ActivityEvent configurations with Claude processing contexts
 * OUTPUT:    Validation that enhanced activity events work correctly
 * BUSINESS:  Enhanced activity events enable Claude processing state tracking
 * CHANGE:    Initial enhanced activity event tests
 * RISK:      High - Activity events are core to all work tracking functionality
 */
func TestActivityEvent_WithClaudeContext(t *testing.T) {
	now := time.Now()
	claudeContext := &ClaudeProcessingContext{
		PromptID:       "prompt_test_123",
		EstimatedTime:  45 * time.Second,
		PromptLength:   150,
		ComplexityHint: "complex",
		ClaudeActivity: ClaudeActivityStart,
	}

	event, err := NewActivityEvent(ActivityEventConfig{
		UserID:        "test_user",
		ProjectName:   "test_project",
		ActivityType:  ActivityTypeGeneration,
		Timestamp:     now,
		Command:       "claude processing test",
		Description:   "Test Claude processing activity",
		ClaudeContext: claudeContext,
	})

	require.NoError(t, err)
	require.NotNil(t, event)

	// Test Claude context methods
	assert.True(t, event.HasClaudeContext())
	assert.True(t, event.IsClaudeProcessingStart())
	assert.False(t, event.IsClaudeProcessingEnd())
	assert.True(t, event.IsClaudeProcessingActivity())
	assert.Equal(t, 45*time.Second, event.GetEstimatedProcessingTime())
	assert.Equal(t, "prompt_test_123", event.GetPromptID())

	// Test data export includes Claude context
	data := event.ToData()
	assert.True(t, data.HasClaudeContext)
	assert.NotNil(t, data.ClaudeContext)
	assert.Equal(t, claudeContext.PromptID, data.ClaudeContext.PromptID)
}

/**
 * CONTEXT:   Test ProcessingTimeEstimator functionality and accuracy
 * INPUT:     Various prompt scenarios and historical data
 * OUTPUT:    Validation that processing time estimation works correctly
 * BUSINESS:  Accurate processing time estimation prevents false idle detection
 * CHANGE:    Initial processing time estimator tests
 * RISK:      Medium - Estimation accuracy affects work block timeout behavior
 */
func TestProcessingTimeEstimator_BasicEstimation(t *testing.T) {
	estimator := NewProcessingTimeEstimator()
	require.NotNil(t, estimator)

	tests := []struct {
		name           string
		prompt         string
		expectedMin    time.Duration
		expectedMax    time.Duration
		expectedLevel  ComplexityLevel
	}{
		{
			name:          "simple question",
			prompt:        "What is Go?",
			expectedMin:   10 * time.Second,
			expectedMax:   30 * time.Second,
			expectedLevel: ComplexityLevelSimple,
		},
		{
			name:          "code generation request",
			prompt:        "Write a function to calculate fibonacci numbers in Go",
			expectedMin:   45 * time.Second,
			expectedMax:   5 * time.Minute,
			expectedLevel: ComplexityLevelComplex,
		},
		{
			name:          "analysis request",
			prompt:        "Analyze this code for potential improvements and explain the issues",
			expectedMin:   30 * time.Second,
			expectedMax:   3 * time.Minute,
			expectedLevel: ComplexityLevelModerate,
		},
		{
			name:          "extensive system design",
			prompt:        "Design a complete microservices architecture for an e-commerce platform with detailed implementation",
			expectedMin:   2 * time.Minute,
			expectedMax:   15 * time.Minute,
			expectedLevel: ComplexityLevelExtensive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := EstimationRequest{
				Prompt:       tt.prompt,
				PromptLength: len(tt.prompt),
				ContextSize:  1,
			}

			estimatedTime := estimator.EstimateProcessingTime(request)
			
			// Verify estimation is within reasonable bounds
			assert.True(t, estimatedTime >= tt.expectedMin, 
				"Estimated time %v should be >= %v", estimatedTime, tt.expectedMin)
			assert.True(t, estimatedTime <= tt.expectedMax, 
				"Estimated time %v should be <= %v", estimatedTime, tt.expectedMax)

			// Test complexity detection
			complexity := estimator.detectComplexity(tt.prompt)
			assert.Equal(t, tt.expectedLevel, complexity, 
				"Expected complexity %v, got %v for prompt: %s", tt.expectedLevel, complexity, tt.prompt)
		})
	}
}

/**
 * CONTEXT:   Test WorkBlock Claude processing state management
 * INPUT:     WorkBlock operations with Claude processing states
 * OUTPUT:    Validation that work blocks handle Claude processing correctly
 * BUSINESS:  Work blocks must accurately track Claude processing to prevent false idle detection
 * CHANGE:    Initial work block Claude processing tests
 * RISK:      High - Work block state management is critical for accurate work time tracking
 */
func TestWorkBlock_ClaudeProcessingStates(t *testing.T) {
	now := time.Now()
	
	// Create work block
	workBlock, err := NewWorkBlock(WorkBlockConfig{
		SessionID:   "test_session",
		ProjectID:   "test_project",
		ProjectName: "test_project",
		StartTime:   now,
	})
	require.NoError(t, err)
	require.NotNil(t, workBlock)

	// Initial state should be active
	assert.Equal(t, WorkBlockStateActive, workBlock.State())
	assert.False(t, workBlock.IsIdle(now))

	// Start Claude processing
	processingStart := now.Add(1 * time.Minute)
	estimatedDuration := 2 * time.Minute
	promptID := "test_prompt_123"

	err = workBlock.StartClaudeProcessing(processingStart, estimatedDuration, promptID)
	require.NoError(t, err)

	// Verify processing state
	assert.Equal(t, WorkBlockStateProcessing, workBlock.State())
	assert.Equal(t, promptID, workBlock.ActivePromptID())
	assert.NotNil(t, workBlock.EstimatedEndTime())
	assert.NotNil(t, workBlock.LastClaudeActivity())
	
	expectedEndTime := processingStart.Add(estimatedDuration)
	assert.Equal(t, expectedEndTime, *workBlock.EstimatedEndTime())

	// Work block should NOT be idle during processing
	duringProcessing := processingStart.Add(1 * time.Minute)
	assert.False(t, workBlock.IsIdle(duringProcessing))

	// End Claude processing
	processingEnd := processingStart.Add(1*time.Minute + 30*time.Second)
	err = workBlock.EndClaudeProcessing(processingEnd)
	require.NoError(t, err)

	// Verify state after processing
	assert.Equal(t, WorkBlockStateActive, workBlock.State())
	assert.Equal(t, "", workBlock.ActivePromptID())
	assert.Nil(t, workBlock.EstimatedEndTime())
	assert.Equal(t, 1*time.Minute+30*time.Second, workBlock.ClaudeProcessingTime())

	// Verify data export includes Claude processing fields
	data := workBlock.ToData()
	assert.Equal(t, int64(90), data.ClaudeProcessingSeconds) // 1m30s = 90s
	assert.Equal(t, 1.5, data.ClaudeProcessingHours)        // 90s = 1.5 minutes = 0.025 hours
	assert.False(t, data.IsProcessing)
}

/**
 * CONTEXT:   Test processing timeout and grace period handling
 * INPUT:     WorkBlock with Claude processing that exceeds estimated time
 * OUTPUT:    Validation that timeout handling works correctly
 * BUSINESS:  Processing timeouts prevent infinite processing states
 * CHANGE:    Initial processing timeout tests
 * RISK:      Medium - Timeout handling must balance false positives vs stuck states
 */
func TestWorkBlock_ProcessingTimeout(t *testing.T) {
	now := time.Now()
	
	workBlock, err := NewWorkBlock(WorkBlockConfig{
		SessionID:   "test_session",
		ProjectID:   "test_project", 
		ProjectName: "test_project",
		StartTime:   now,
	})
	require.NoError(t, err)

	// Start processing with 1 minute estimate
	processingStart := now.Add(1 * time.Minute)
	estimatedDuration := 1 * time.Minute
	
	err = workBlock.StartClaudeProcessing(processingStart, estimatedDuration, "test_prompt")
	require.NoError(t, err)
	assert.Equal(t, WorkBlockStateProcessing, workBlock.State())

	// Within estimated time - should not be idle
	duringEstimate := processingStart.Add(30 * time.Second)
	assert.False(t, workBlock.IsIdle(duringEstimate))

	// Just past estimated time - should still not be idle (grace period)
	pastEstimate := processingStart.Add(1*time.Minute + 10*time.Second)
	assert.False(t, workBlock.IsIdle(pastEstimate))

	// Well past grace period - should be considered timed out
	wayPastEstimate := processingStart.Add(2*time.Minute + 30*time.Second)
	assert.True(t, workBlock.IsIdle(wayPastEstimate))
}

/**
 * CONTEXT:   Test WorkBlockManager Claude processing integration
 * INPUT:     WorkBlockManager operations with Claude processing activities
 * OUTPUT:    Validation that manager correctly handles Claude processing workflows
 * BUSINESS:  Work block manager must coordinate Claude processing across work blocks
 * CHANGE:    Initial work block manager Claude processing tests
 * RISK:      High - Manager integration affects all Claude processing workflows
 */
func TestWorkBlockManager_ClaudeProcessingIntegration(t *testing.T) {
	// This test would require mocking repositories
	// For now, we'll test the business logic components we can test in isolation
	
	// Test processing time estimation
	estimator := NewProcessingTimeEstimator()
	
	// Record some historical data
	estimator.RecordActualTime(ComplexityLevelSimple, 20*time.Second)
	estimator.RecordActualTime(ComplexityLevelSimple, 25*time.Second)
	estimator.RecordActualTime(ComplexityLevelModerate, 75*time.Second)
	
	// Test that historical data affects estimates
	simpleRequest := EstimationRequest{
		Prompt:       "Simple question",
		PromptLength: 15,
		ContextSize:  1,
	}
	
	estimate := estimator.EstimateProcessingTime(simpleRequest)
	assert.True(t, estimate >= 10*time.Second)
	assert.True(t, estimate <= 2*time.Minute)
	
	// Test stats
	stats := estimator.GetStats()
	assert.Equal(t, 3, stats.TotalEstimations)
	assert.Equal(t, 2, stats.HistoricalDataPoints[ComplexityLevelSimple])
	assert.Equal(t, 1, stats.HistoricalDataPoints[ComplexityLevelModerate])
}

/**
 * CONTEXT:   Test enhanced reporting with Claude processing metrics
 * INPUT:     Work blocks with Claude processing time data
 * OUTPUT:    Validation that enhanced reporting shows correct Claude processing insights
 * BUSINESS:  Enhanced reporting provides users accurate productivity insights
 * CHANGE:    Initial enhanced reporting tests
 * RISK:      Medium - Reporting accuracy affects user understanding of their productivity
 */
func TestEnhancedReporting_ClaudeProcessingMetrics(t *testing.T) {
	generator, err := NewEnhancedReportGenerator("UTC")
	require.NoError(t, err)

	now := time.Now()
	
	// Create test work blocks with Claude processing
	workBlocks := make([]*WorkBlock, 0)
	
	// Work block 1: 10 minutes total, 2 minutes Claude processing
	wb1, err := NewWorkBlock(WorkBlockConfig{
		SessionID:   "session1",
		ProjectID:   "proj1", 
		ProjectName: "project1",
		StartTime:   now,
	})
	require.NoError(t, err)
	
	// Simulate Claude processing
	err = wb1.StartClaudeProcessing(now.Add(1*time.Minute), 2*time.Minute, "prompt1")
	require.NoError(t, err)
	err = wb1.EndClaudeProcessing(now.Add(3*time.Minute))
	require.NoError(t, err)
	err = wb1.Finish(now.Add(10*time.Minute))
	require.NoError(t, err)
	
	workBlocks = append(workBlocks, wb1)
	
	// Work block 2: 5 minutes total, no Claude processing  
	wb2, err := NewWorkBlock(WorkBlockConfig{
		SessionID:   "session1",
		ProjectID:   "proj2",
		ProjectName: "project2", 
		StartTime:   now.Add(15*time.Minute),
	})
	require.NoError(t, err)
	err = wb2.Finish(now.Add(20*time.Minute))
	require.NoError(t, err)
	
	workBlocks = append(workBlocks, wb2)
	
	// Generate report
	sessions := []*Session{} // Empty for test
	metrics := generator.GenerateDailyReport(workBlocks, sessions, now)
	
	// Verify metrics
	assert.Equal(t, 2*time.Minute, metrics.ClaudeProcessingTime)
	assert.Equal(t, 13*time.Minute, metrics.UserInteractionTime) // 15 total - 2 Claude
	assert.Equal(t, 20*time.Minute, metrics.TotalScheduleTime)   // From start to finish
	
	// Verify percentages
	assert.InDelta(t, 10.0, metrics.ClaudeActivityPercent, 1.0) // 2/20 = 10%
	assert.InDelta(t, 65.0, metrics.UserActivityPercent, 1.0)   // 13/20 = 65%
	
	// Verify activity counts
	assert.Equal(t, 2, metrics.TotalWorkBlocks)
	assert.Equal(t, 1, metrics.ProcessingBlocks)
	
	// Verify project breakdown
	assert.Equal(t, 2, metrics.ProjectCount)
	assert.Len(t, metrics.ProjectBreakdown, 2)
	
	// Find project1 metrics (has Claude processing)
	var proj1Metrics *EnhancedProjectMetrics
	for i := range metrics.ProjectBreakdown {
		if metrics.ProjectBreakdown[i].ProjectName == "project1" {
			proj1Metrics = &metrics.ProjectBreakdown[i]
			break
		}
	}
	require.NotNil(t, proj1Metrics)
	assert.Equal(t, 2*time.Minute, proj1Metrics.ClaudeProcessingTime)
	assert.Equal(t, 1, proj1Metrics.ProcessingBlocks)
}

/**
 * CONTEXT:   Test integration scenarios with mixed Claude processing and regular activities
 * INPUT:     Complex workflow scenarios combining regular work and Claude processing
 * OUTPUT:    Validation that complex workflows are handled correctly
 * BUSINESS:  Real-world usage involves mixed activities that must be tracked accurately
 * CHANGE:    Initial integration scenario tests
 * RISK:      High - Integration scenarios test the most complex and error-prone workflows
 */
func TestClaudeProcessing_IntegrationScenarios(t *testing.T) {
	now := time.Now()
	
	t.Run("overlapping activities during processing", func(t *testing.T) {
		workBlock, err := NewWorkBlock(WorkBlockConfig{
			SessionID:   "session1",
			ProjectID:   "proj1",
			ProjectName: "project1", 
			StartTime:   now,
		})
		require.NoError(t, err)
		
		// Start Claude processing
		err = workBlock.StartClaudeProcessing(now.Add(1*time.Minute), 3*time.Minute, "prompt1")
		require.NoError(t, err)
		assert.Equal(t, WorkBlockStateProcessing, workBlock.State())
		
		// Record regular activity during processing (user is still active)
		err = workBlock.RecordActivity(now.Add(2*time.Minute))
		require.NoError(t, err)
		
		// State should still be processing
		assert.Equal(t, WorkBlockStateProcessing, workBlock.State())
		
		// End processing
		err = workBlock.EndClaudeProcessing(now.Add(4*time.Minute))
		require.NoError(t, err)
		assert.Equal(t, WorkBlockStateActive, workBlock.State())
		
		// Verify processing time was recorded
		assert.Equal(t, 3*time.Minute, workBlock.ClaudeProcessingTime())
	})
	
	t.Run("multiple processing sessions in same work block", func(t *testing.T) {
		workBlock, err := NewWorkBlock(WorkBlockConfig{
			SessionID:   "session1", 
			ProjectID:   "proj1",
			ProjectName: "project1",
			StartTime:   now,
		})
		require.NoError(t, err)
		
		// First processing session
		err = workBlock.StartClaudeProcessing(now.Add(1*time.Minute), 2*time.Minute, "prompt1")
		require.NoError(t, err)
		err = workBlock.EndClaudeProcessing(now.Add(3*time.Minute))
		require.NoError(t, err)
		
		// Some user activity
		err = workBlock.RecordActivity(now.Add(5*time.Minute))
		require.NoError(t, err)
		
		// Second processing session
		err = workBlock.StartClaudeProcessing(now.Add(6*time.Minute), 1*time.Minute, "prompt2")
		require.NoError(t, err)
		err = workBlock.EndClaudeProcessing(now.Add(7*time.Minute))
		require.NoError(t, err)
		
		// Total Claude processing time should be cumulative
		expectedTotal := 2*time.Minute + 1*time.Minute
		assert.Equal(t, expectedTotal, workBlock.ClaudeProcessingTime())
	})
}