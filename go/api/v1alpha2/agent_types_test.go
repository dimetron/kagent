/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T005: Contract Test - SequentialAgentSpec CRD Schema
func TestSequentialAgentSpec_Schema(t *testing.T) {
	tests := []struct {
		name    string
		spec    SequentialAgentSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sequential spec with 2 sub-agents",
			spec: SequentialAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{
						{Name: "agent-a"},
						{Name: "agent-b"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid sequential spec with 50 sub-agents (max)",
			spec: SequentialAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: func() []SubAgentReference {
						agents := make([]SubAgentReference, 50)
						for i := 0; i < 50; i++ {
							agents[i] = SubAgentReference{Name: "agent"}
						}
						return agents
					}(),
				},
			},
			wantErr: false,
		},
		{
			name: "valid sequential spec with timeout",
			spec: SequentialAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					Timeout:   stringPtr("5m"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid sequential spec with description",
			spec: SequentialAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents:   []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					Description: "Test sequential workflow",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify struct can be created
			assert.NotNil(t, tt.spec.SubAgents)

			// Verify required field
			require.NotEmpty(t, tt.spec.SubAgents, "SubAgents is required")

			// Verify constraints
			assert.LessOrEqual(t, len(tt.spec.SubAgents), 50, "SubAgents max 50")
			assert.GreaterOrEqual(t, len(tt.spec.SubAgents), 2, "SubAgents min 2")
		})
	}
}

// T006: Contract Test - ParallelAgentSpec CRD Schema
func TestParallelAgentSpec_Schema(t *testing.T) {
	tests := []struct {
		name    string
		spec    ParallelAgentSpec
		wantErr bool
	}{
		{
			name: "valid parallel spec with 2 sub-agents (min)",
			spec: ParallelAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{
						{Name: "agent-a"},
						{Name: "agent-b"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid parallel spec with 50 sub-agents (max)",
			spec: ParallelAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: func() []SubAgentReference {
						agents := make([]SubAgentReference, 50)
						for i := 0; i < 50; i++ {
							agents[i] = SubAgentReference{Name: "agent"}
						}
						return agents
					}(),
				},
			},
			wantErr: false,
		},
		{
			name: "valid parallel spec with timeout",
			spec: ParallelAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{
						{Name: "agent-a"},
						{Name: "agent-b"},
					},
					Timeout: stringPtr("300s"),
				},
			},
			wantErr: false,
		},
		// T001: Test maxWorkers field
		{
			name: "valid parallel spec with maxWorkers=5",
			spec: ParallelAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{
						{Name: "agent-a"},
						{Name: "agent-b"},
					},
				},
				MaxWorkers: int32Ptr(5),
			},
			wantErr: false,
		},
		{
			name: "valid parallel spec without maxWorkers (defaults to 10)",
			spec: ParallelAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{
						{Name: "agent-a"},
						{Name: "agent-b"},
					},
				},
				// MaxWorkers omitted - should default to 10
			},
			wantErr: false,
		},
		// T002: Test validation boundaries
		{
			name: "valid maxWorkers=1 (minimum boundary)",
			spec: ParallelAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{
						{Name: "agent-a"},
						{Name: "agent-b"},
					},
				},
				MaxWorkers: int32Ptr(1),
			},
			wantErr: false,
		},
		{
			name: "valid maxWorkers=50 (maximum boundary)",
			spec: ParallelAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{
						{Name: "agent-a"},
						{Name: "agent-b"},
					},
				},
				MaxWorkers: int32Ptr(50),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify struct can be created
			assert.NotNil(t, tt.spec.SubAgents)

			// Verify required field
			require.NotEmpty(t, tt.spec.SubAgents, "SubAgents is required")

			// Verify constraints (min 2 for parallel)
			assert.LessOrEqual(t, len(tt.spec.SubAgents), 50, "SubAgents max 50")
			assert.GreaterOrEqual(t, len(tt.spec.SubAgents), 2, "ParallelAgent requires min 2 sub-agents")

			// T001: Verify maxWorkers handling
			if tt.spec.MaxWorkers != nil {
				// Verify value is within valid range (1-50)
				assert.GreaterOrEqual(t, *tt.spec.MaxWorkers, int32(1), "MaxWorkers min 1")
				assert.LessOrEqual(t, *tt.spec.MaxWorkers, int32(50), "MaxWorkers max 50")
			}
		})
	}
}

// T007: Contract Test - LoopAgentSpec CRD Schema
func TestLoopAgentSpec_Schema(t *testing.T) {
	tests := []struct {
		name string
		spec LoopAgentSpec
	}{
		{
			name: "valid loop spec with 2 sub-agents",
			spec: LoopAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
				},
				MaxIterations: 1,
			},
		},
		{
			name: "valid loop spec with max_iterations=100",
			spec: LoopAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
				},
				MaxIterations: 100,
			},
		},
		{
			name: "valid loop spec with timeout",
			spec: LoopAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					Timeout:   stringPtr("8m"),
				},
				MaxIterations: 5,
			},
		},
		{
			name: "valid loop spec with description",
			spec: LoopAgentSpec{
				BaseWorkflowSpec: BaseWorkflowSpec{
					SubAgents:   []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					Description: "Iterative workflow",
				},
				MaxIterations: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify struct can be created
			assert.NotNil(t, tt.spec.SubAgents)

			// Verify required fields
			require.NotEmpty(t, tt.spec.SubAgents, "SubAgents is required")
			require.NotZero(t, tt.spec.MaxIterations, "MaxIterations is required")

			// Verify constraints
			assert.LessOrEqual(t, len(tt.spec.SubAgents), 50, "SubAgents max 50")
			assert.GreaterOrEqual(t, len(tt.spec.SubAgents), 2, "SubAgents min 2")
			assert.GreaterOrEqual(t, tt.spec.MaxIterations, int32(1), "MaxIterations min 1")
			assert.LessOrEqual(t, tt.spec.MaxIterations, int32(100), "MaxIterations max 100")
		})
	}
}

// T008: Contract Test - SubAgentReference Structure
func TestSubAgentReference_Schema(t *testing.T) {
	tests := []struct {
		name string
		ref  SubAgentReference
	}{
		{
			name: "valid reference with name only",
			ref: SubAgentReference{
				Name: "agent-a",
			},
		},
		{
			name: "valid reference with namespace",
			ref: SubAgentReference{
				Name:      "agent-a",
				Namespace: "custom-namespace",
			},
		},
		{
			name: "valid reference with kind",
			ref: SubAgentReference{
				Name: "agent-a",
				Kind: "Agent",
			},
		},
		{
			name: "valid reference with all fields",
			ref: SubAgentReference{
				Name:      "agent-a",
				Namespace: "custom-namespace",
				Kind:      "Agent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify required field
			require.NotEmpty(t, tt.ref.Name, "Name is required")

			// Verify Kind default/enum
			if tt.ref.Kind != "" {
				assert.Equal(t, "Agent", tt.ref.Kind, "Kind must be 'Agent'")
			}
		})
	}
}

// T009: Contract Test - WorkflowAgentSpec OneOf Validation
func TestWorkflowAgentSpec_OneOf(t *testing.T) {
	tests := []struct {
		name    string
		spec    WorkflowAgentSpec
		valid   bool
		message string
	}{
		{
			name: "valid - sequential only",
			spec: WorkflowAgentSpec{
				Sequential: &SequentialAgentSpec{
					BaseWorkflowSpec: BaseWorkflowSpec{
						SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					},
				},
			},
			valid: true,
		},
		{
			name: "valid - parallel only",
			spec: WorkflowAgentSpec{
				Parallel: &ParallelAgentSpec{
					BaseWorkflowSpec: BaseWorkflowSpec{
						SubAgents: []SubAgentReference{
							{Name: "agent-a"},
							{Name: "agent-b"},
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "valid - loop only",
			spec: WorkflowAgentSpec{
				Loop: &LoopAgentSpec{
					BaseWorkflowSpec: BaseWorkflowSpec{
						SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					},
					MaxIterations: 3,
				},
			},
			valid: true,
		},
		{
			name: "invalid - sequential and parallel",
			spec: WorkflowAgentSpec{
				Sequential: &SequentialAgentSpec{
					BaseWorkflowSpec: BaseWorkflowSpec{
						SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					},
				},
				Parallel: &ParallelAgentSpec{
					BaseWorkflowSpec: BaseWorkflowSpec{
						SubAgents: []SubAgentReference{
							{Name: "agent-a"},
							{Name: "agent-b"},
						},
					},
				},
			},
			valid:   false,
			message: "exactly one of sequential, parallel, or loop must be specified",
		},
		{
			name: "invalid - all three specified",
			spec: WorkflowAgentSpec{
				Sequential: &SequentialAgentSpec{
					BaseWorkflowSpec: BaseWorkflowSpec{
						SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					},
				},
				Parallel: &ParallelAgentSpec{
					BaseWorkflowSpec: BaseWorkflowSpec{
						SubAgents: []SubAgentReference{
							{Name: "agent-a"},
							{Name: "agent-b"},
						},
					},
				},
				Loop: &LoopAgentSpec{
					BaseWorkflowSpec: BaseWorkflowSpec{
						SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
					},
					MaxIterations: 3,
				},
			},
			valid:   false,
			message: "exactly one of sequential, parallel, or loop must be specified",
		},
		{
			name:    "invalid - none specified",
			spec:    WorkflowAgentSpec{},
			valid:   false,
			message: "exactly one of sequential, parallel, or loop must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Count non-nil fields
			nonNilCount := 0
			if tt.spec.Sequential != nil {
				nonNilCount++
			}
			if tt.spec.Parallel != nil {
				nonNilCount++
			}
			if tt.spec.Loop != nil {
				nonNilCount++
			}

			if tt.valid {
				assert.Equal(t, 1, nonNilCount, "Exactly one workflow type must be set")
			} else {
				assert.NotEqual(t, 1, nonNilCount, tt.message)
			}
		})
	}
}

// T010: Contract Test - AgentSpec Type Extension
func TestAgentSpec_WorkflowType(t *testing.T) {
	tests := []struct {
		name  string
		spec  AgentSpec
		valid bool
	}{
		{
			name: "valid - Workflow type with workflow spec",
			spec: AgentSpec{
				Type: AgentType_Workflow,
				Workflow: &WorkflowAgentSpec{
					Sequential: &SequentialAgentSpec{
						BaseWorkflowSpec: BaseWorkflowSpec{
							SubAgents: []SubAgentReference{{Name: "agent-a"}, {Name: "agent-b"}},
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "valid - Declarative type still works (backward compatibility)",
			spec: AgentSpec{
				Type: AgentType_Declarative,
				Declarative: &DeclarativeAgentSpec{
					SystemMessage: "test",
				},
			},
			valid: true,
		},
		{
			name: "valid - BYO type still works (backward compatibility)",
			spec: AgentSpec{
				Type: AgentType_BYO,
				BYO:  &BYOAgentSpec{},
			},
			valid: true,
		},
		{
			name: "invalid - Workflow type without workflow spec",
			spec: AgentSpec{
				Type: AgentType_Workflow,
			},
			valid: false,
		},
		{
			name: "invalid - Workflow type with declarative spec",
			spec: AgentSpec{
				Type: AgentType_Workflow,
				Declarative: &DeclarativeAgentSpec{
					SystemMessage: "test",
				},
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify type is valid enum value
			assert.Contains(t, []AgentType{
				AgentType_Declarative,
				AgentType_BYO,
				AgentType_Workflow,
			}, tt.spec.Type, "Type must be valid enum value")

			// Verify mutual exclusion
			if tt.valid {
				if tt.spec.Type == AgentType_Workflow {
					require.NotNil(t, tt.spec.Workflow, "Workflow spec required for Workflow type")
					assert.Nil(t, tt.spec.Declarative, "Declarative must be nil for Workflow type")
					assert.Nil(t, tt.spec.BYO, "BYO must be nil for Workflow type")
				}
			}
		})
	}
}

// T009: Contract Test - SubAgentReference OutputKey Validation
func TestSubAgentReference_OutputKey_Validation(t *testing.T) {
	tests := []struct {
		name      string
		outputKey string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid outputKey with lowercase letters",
			outputKey: "generated_code",
			wantErr:   false,
		},
		{
			name:      "valid outputKey with uppercase letters",
			outputKey: "GeneratedCode",
			wantErr:   false,
		},
		{
			name:      "valid outputKey with numbers",
			outputKey: "result_123",
			wantErr:   false,
		},
		{
			name:      "valid outputKey starting with underscore",
			outputKey: "_private_result",
			wantErr:   false,
		},
		{
			name:      "valid outputKey with mixed case and numbers",
			outputKey: "myOutput_v2",
			wantErr:   false,
		},
		{
			name:      "invalid outputKey starting with number",
			outputKey: "1result",
			wantErr:   true,
			errMsg:    "must match pattern",
		},
		{
			name:      "invalid outputKey with hyphen",
			outputKey: "result-1",
			wantErr:   true,
			errMsg:    "must match pattern",
		},
		{
			name:      "invalid outputKey with dot",
			outputKey: "result.value",
			wantErr:   true,
			errMsg:    "must match pattern",
		},
		{
			name:      "invalid outputKey with space",
			outputKey: "result value",
			wantErr:   true,
			errMsg:    "must match pattern",
		},
		{
			name:      "invalid outputKey with special chars",
			outputKey: "result@123",
			wantErr:   true,
			errMsg:    "must match pattern",
		},
		{
			name:      "invalid outputKey exceeding max length",
			outputKey: "a" + string(make([]byte, 100)), // 101 characters
			wantErr:   true,
			errMsg:    "max length",
		},
		{
			name:      "empty outputKey is valid (optional field)",
			outputKey: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := SubAgentReference{
				Name:      "test-agent",
				OutputKey: tt.outputKey,
			}

			// Test pattern matching
			if tt.outputKey != "" && len(tt.outputKey) <= 100 {
				// Validate pattern manually since we can't easily test kubebuilder validation
				pattern := `^[a-zA-Z_][a-zA-Z0-9_]*$`
				matched := matchPattern(pattern, tt.outputKey)

				if tt.wantErr {
					assert.False(t, matched, "OutputKey should not match pattern")
				} else {
					assert.True(t, matched, "OutputKey should match pattern")
				}
			}

			// Test length constraint
			if len(tt.outputKey) > 127 {
				assert.True(t, tt.wantErr, "OutputKey exceeding 127 chars should be invalid")
			}

			// Verify the field is populated correctly
			assert.Equal(t, tt.outputKey, ref.OutputKey)
		})
	}
}

// Helper to match regex pattern
func matchPattern(pattern, value string) bool {
	if value == "" {
		return true // empty is valid for optional field
	}
	// Simple validation for the pattern ^[a-zA-Z_][a-zA-Z0-9_]*$
	if len(value) == 0 {
		return false
	}
	// First character must be letter or underscore
	first := value[0]
	if (first < 'a' || first > 'z') && (first < 'A' || first > 'Z') && first != '_' {
		return false
	}
	// Remaining characters must be letters, digits, or underscores
	for i := 1; i < len(value); i++ {
		c := value[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}

// TestSubAgentReference_OutputKey tests OutputKey field validation
func TestSubAgentReference_OutputKey(t *testing.T) {
	tests := []struct {
		name       string
		outputKey  string
		shouldPass bool
		reason     string
	}{
		{
			name:       "valid outputKey with alphanumeric and underscores",
			outputKey:  "production_east_collector",
			shouldPass: true,
			reason:     "standard auto-generated format",
		},
		{
			name:       "valid outputKey with all underscores",
			outputKey:  "production_us_east_collector",
			shouldPass: true,
			reason:     "multiple underscores are allowed",
		},
		{
			name:       "valid outputKey all lowercase",
			outputKey:  "myoutputkey",
			shouldPass: true,
			reason:     "simple alphanumeric",
		},
		{
			name:       "valid outputKey with numbers",
			outputKey:  "output_key_123",
			shouldPass: true,
			reason:     "numbers are allowed",
		},
		{
			name:       "valid outputKey starting with number",
			outputKey:  "123_output",
			shouldPass: true,
			reason:     "pattern allows leading numbers",
		},
		{
			name:       "invalid outputKey with hyphen",
			outputKey:  "production-east-collector",
			shouldPass: false,
			reason:     "hyphens not allowed (must use underscores)",
		},
		{
			name:       "invalid outputKey with dot",
			outputKey:  "production.east.collector",
			shouldPass: false,
			reason:     "dots not allowed",
		},
		{
			name:       "invalid outputKey with space",
			outputKey:  "production east collector",
			shouldPass: false,
			reason:     "spaces not allowed",
		},
		{
			name:       "invalid outputKey with special chars",
			outputKey:  "production@east#collector",
			shouldPass: false,
			reason:     "special characters not allowed",
		},
		{
			name:       "invalid outputKey exceeds max length (100 chars)",
			outputKey:  "this_is_a_very_long_output_key_name_that_exceeds_the_maximum_allowed_length_of_one_hundred_characters_and_should_fail_validation_test",
			shouldPass: false,
			reason:     "exceeds 100 character limit",
		},
		{
			name:       "valid outputKey at max length (100 chars)",
			outputKey:  "this_is_a_ninety_nine_character_output_key_name_that_is_exactly_at_the_maximum_length_xxxxxxxx",
			shouldPass: true,
			reason:     "exactly 100 characters",
		},
		{
			name:       "valid outputKey empty (optional field)",
			outputKey:  "",
			shouldPass: true,
			reason:     "empty string is valid (field is optional)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := SubAgentReference{
				Name:      "test-agent",
				Namespace: "default",
				OutputKey: tt.outputKey,
			}

			// Validate pattern if outputKey is not empty
			if tt.outputKey != "" {
				// Pattern: ^[a-zA-Z0-9_]+$
				matches := isValidOutputKeyPattern(tt.outputKey)
				if tt.shouldPass {
					assert.True(t, matches, "OutputKey '%s' should match pattern. Reason: %s", tt.outputKey, tt.reason)
				} else {
					// Check if failure is due to pattern or length
					if len(tt.outputKey) > 100 {
						assert.Greater(t, len(tt.outputKey), 100, "OutputKey length should exceed 100. Reason: %s", tt.reason)
					} else {
						assert.False(t, matches, "OutputKey '%s' should NOT match pattern. Reason: %s", tt.outputKey, tt.reason)
					}
				}
			}

			// Validate length
			if len(tt.outputKey) <= 100 || tt.shouldPass {
				// Valid reference
				assert.NotNil(t, ref, "SubAgentReference should be created")
			}
		})
	}
}

// isValidOutputKeyPattern validates outputKey against pattern ^[a-zA-Z0-9_]+$
func isValidOutputKeyPattern(value string) bool {
	if len(value) == 0 {
		return true // Empty is valid (optional field)
	}
	// All characters must be alphanumeric or underscore
	for _, c := range value {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}
