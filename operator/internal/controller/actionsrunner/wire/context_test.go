package wire

import (
	"encoding/json"
	"testing"
)

const (
	contextData = `
		{
			"github": {
				"t": 2,
				"d": [
					{
						"k": "ref",
						"v": "refs/heads/master"
					},
					{
						"k": "event",
						"v": {
							"t": 2,
							"d": [
								{
									"k": "organization",
									"v": {
										"t": 2,
										"d": [
											{
												"k": "id",
												"v": 33351090.0
											},
											{
												"k": "login",
												"v": "inloco"
											}
										]
									}
								},
								{
									"k": "repository",
									"v": {
										"t": 2,
										"d": [
											{
												"k": "archived",
												"v": false
											},
											{
												"k": "default_branch",
												"v": "master"
											},
											{
												"k": "forks",
												"v": 0.0
											},
											{
												"k": "has_downloads",
												"v": true
											},
											{
												"k": "homepage",
												"v": null
											},
											{
												"k": "open_issues",
												"v": 2.0
											},
											{
												"k": "owner",
												"v": {
													"t": 2,
													"d": [
														{
															"k": "gravatar_id",
															"v": ""
														},
														{
															"k": "html_url",
															"v": "https://github.com/inloco"
														},
														{
															"k": "site_admin",
															"v": false
														}
													]
												}
											},
											{
												"k": "topics",
												"v": {
													"t": 1,
													"a": [
														"hacktoberfest"
													]
												}
											}
										]
									}
								},
								{
									"k": "schedule",
									"v": "* * * * *"
								}
							]
						}
					},
					{
						"k": "ref_protected",
						"v": true
					}
				]
			},
			"run-feature-flags": {
				"t": 2,
				"d": [
					{
						"k": "actions_use_results_service",
						"v": false
					}
				]
			},
			"partialrerun": false,
			"needs": {
				"t": 2
			},
			"vars": null,
			"inputs": {
				"t": 2
			},
			"matrix": null,
			"strategy": {
				"t": 2,
				"d": [
					{
						"k": "fail-fast",
						"v": true
					},
					{
						"k": "job-index",
						"v": 0.0
					},
					{
						"k": "job-total",
						"v": 1.0
					}
				]
			}
		}
	`
)

func TestPipelineContextDataUnmarshalJSON(t *testing.T) {
	var pcds map[string]PipelineContextData
	if err := json.Unmarshal([]byte(contextData), &pcds); err != nil {
		t.Error(err)
	}

	github, ok := pcds["github"]
	if !ok {
		t.Error(`pcds["github"] == nil`)
	}

	if github.Type != PipelineContextDataTypeDictionary {
		t.Error(`github.Type != PipelineContextDataTypeDictionary`)
	}

	if len(github.Dictionary) != 3 {
		t.Error(`len(github.Dictionary) != 3`)
	}

	if github.Dictionary[0].Key != "ref" {
		t.Error(`github.Dictionary[0].Key != "ref"`)
	}

	if github.Dictionary[0].Val.Type != PipelineContextDataTypeString {
		t.Error(`github.Dictionary[0].Val.Type != PipelineContextDataTypeString`)
	}

	if github.Dictionary[0].Val.String != "refs/heads/master" {
		t.Error(`github.Dictionary[0].Val.String != "refs/heads/master"`)
	}

	if github.Dictionary[1].Key != "event" {
		t.Error(`github.Dictionary[1].Key != "event"`)
	}

	if github.Dictionary[1].Val.Type != PipelineContextDataTypeDictionary {
		t.Error(`github.Dictionary[1].Val.Type != PipelineContextDataTypeDictionary`)
	}

	if len(github.Dictionary[1].Val.Dictionary) != 3 {
		t.Error(`len(github.Dictionary[1].Val.Dictionary) != 3`)
	}

	if github.Dictionary[1].Val.Dictionary[0].Key != "organization" {
		t.Error(`github.Dictionary[1].Val.Dictionary[0].Key != "organization"`)
	}

	if github.Dictionary[1].Val.Dictionary[0].Val.Type != PipelineContextDataTypeDictionary {
		t.Error(`github.Dictionary[1].Val.Dictionary[0].Val.Type != PipelineContextDataTypeDictionary`)
	}

	if len(github.Dictionary[1].Val.Dictionary[0].Val.Dictionary) != 2 {
		t.Error(`len(github.Dictionary[1].Val.Dictionary[0].Val.Dictionary) != 2`)
	}

	if github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[0].Key != "id" {
		t.Error(`github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[0].Key != "id"`)
	}

	if github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[0].Val.Type != PipelineContextDataTypeNumber {
		t.Error(`github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[0].Val.Type != PipelineContextDataTypeNumber`)
	}

	if github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[0].Val.Number != 33351090. {
		t.Error(`github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[0].Val.Number != 33351090.`)
	}

	if github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[1].Key != "login" {
		t.Error(`github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[1].Key != "login"`)
	}

	if github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[1].Val.Type != PipelineContextDataTypeString {
		t.Error(`github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[1].Val.Type != PipelineContextDataTypeString`)
	}

	if github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[1].Val.String != "inloco" {
		t.Error(`github.Dictionary[1].Val.Dictionary[0].Val.Dictionary[1].Val.String != "inloco"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Key != "repository" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Key != "repository"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Type != PipelineContextDataTypeDictionary {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Type != PipelineContextDataTypeDictionary`)
	}

	if len(github.Dictionary[1].Val.Dictionary[1].Val.Dictionary) != 8 {
		t.Error(`len(github.Dictionary[1].Val.Dictionary[1].Val.Dictionary) != 8`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[0].Key != "archived" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[0].Key != "archived"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[0].Val.Type != PipelineContextDataTypeBoolean {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[0].Val.Type != PipelineContextDataTypeBoolean`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[0].Val.Boolean != false {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[0].Val.Boolean != false`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[1].Key != "default_branch" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[1].Key != "default_branch"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[1].Val.Type != PipelineContextDataTypeString {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[1].Val.Type != PipelineContextDataTypeString`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[1].Val.String != "master" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[1].Val.String != "master"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[2].Key != "forks" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[2].Key != "forks"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[2].Val.Type != PipelineContextDataTypeNumber {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[2].Val.Type != PipelineContextDataTypeNumber`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[2].Val.Number != .0 {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[2].Val.Number != .0`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[3].Key != "has_downloads" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[3].Key != "has_downloads"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[3].Val.Type != PipelineContextDataTypeBoolean {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[3].Val.Type != PipelineContextDataTypeBoolean`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[3].Val.Boolean != true {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[3].Val.Boolean != true`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[4].Key != "homepage" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[4].Key != "homepage"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[4].Val.Type != PipelineContextDataTypeString {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[4].Val.Type != PipelineContextDataTypeString`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[4].Val.String != "" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[4].Val.String != ""`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[5].Key != "open_issues" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[5].Key != "open_issues"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[5].Val.Type != PipelineContextDataTypeNumber {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[5].Val.Type != PipelineContextDataTypeNumber`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[5].Val.Number != 2. {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[5].Val.Number != 2.`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Key != "owner" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Key != "owner"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Type != PipelineContextDataTypeDictionary {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Type != PipelineContextDataTypeDictionary`)
	}

	if len(github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary) != 3 {
		t.Error(`len(github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary) != 3`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[0].Key != "gravatar_id" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[0].Key != "gravatar_id"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[0].Val.Type != PipelineContextDataTypeString {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[0].Val.Type != PipelineContextDataTypeString`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[0].Val.String != "" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[0].Val.String != ""`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[1].Key != "html_url" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[1].Key != "html_url"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[1].Val.Type != PipelineContextDataTypeString {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[1].Val.Type != PipelineContextDataTypeString`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[1].Val.String != "https://github.com/inloco" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[1].Val.String != "https://github.com/inloco"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[2].Key != "site_admin" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[2].Key != "site_admin"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[2].Val.Type != PipelineContextDataTypeBoolean {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[2].Val.Type != PipelineContextDataTypeBoolean`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[2].Val.Boolean != false {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[6].Val.Dictionary[2].Val.Boolean != false`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Key != "topics" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Key != "topics"`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Val.Type != PipelineContextDataTypeArray {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Val.Type != PipelineContextDataTypeArray`)
	}

	if len(github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Val.Array) != 1 {
		t.Error(`len(github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Val.Array) != 1`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Val.Array[0].Type != PipelineContextDataTypeString {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Val.Array[0].Type != PipelineContextDataTypeString`)
	}

	if github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Val.Array[0].String != "hacktoberfest" {
		t.Error(`github.Dictionary[1].Val.Dictionary[1].Val.Dictionary[7].Val.Array[0].String != "hacktoberfest"`)
	}

	if github.Dictionary[1].Val.Dictionary[2].Key != "schedule" {
		t.Error(`github.Dictionary[1].Val.Dictionary[2].Key != "schedule"`)
	}

	if github.Dictionary[1].Val.Dictionary[2].Val.Type != PipelineContextDataTypeString {
		t.Error(`github.Dictionary[1].Val.Dictionary[2].Val.Type != PipelineContextDataTypeString`)
	}

	if github.Dictionary[1].Val.Dictionary[2].Val.String != "* * * * *" {
		t.Error(`github.Dictionary[1].Val.Dictionary[2].Val.String != "* * * * *"`)
	}

	if github.Dictionary[2].Key != "ref_protected" {
		t.Error(`github.Dictionary[2].Key != "ref_protected"`)
	}

	if github.Dictionary[2].Val.Type != PipelineContextDataTypeBoolean {
		t.Error(`github.Dictionary[2].Val.Type != PipelineContextDataTypeBoolean`)
	}

	if github.Dictionary[2].Val.Boolean != true {
		t.Error(`github.Dictionary[2].Val.Boolean != true`)
	}

	runFeatureFlags, ok := pcds["run-feature-flags"]
	if !ok {
		t.Error(`pcds["run-feature-flags"] == nil`)
	}

	if runFeatureFlags.Type != PipelineContextDataTypeDictionary {
		t.Error(`runFeatureFlags.Type != PipelineContextDataTypeDictionary`)
	}

	if len(runFeatureFlags.Dictionary) != 1 {
		t.Error(`len(runFeatureFlags.Dictionary) != 1`)
	}

	if runFeatureFlags.Dictionary[0].Key != "actions_use_results_service" {
		t.Error(`runFeatureFlags.Dictionary[0].Key != "actions_use_results_service"`)
	}

	if runFeatureFlags.Dictionary[0].Val.Type != PipelineContextDataTypeBoolean {
		t.Error(`runFeatureFlags.Dictionary[0].Val.Type != PipelineContextDataTypeBoolean`)
	}

	if runFeatureFlags.Dictionary[0].Val.Boolean != false {
		t.Error(`runFeatureFlags.Dictionary[0].Val.Boolean != false`)
	}

	partialrerun, ok := pcds["partialrerun"]
	if !ok {
		t.Error(`pcds["partialrerun"] == nil`)
	}

	if partialrerun.Type != PipelineContextDataTypeBoolean {
		t.Error(`partialrerun.Type != PipelineContextDataTypeBoolean`)
	}

	if partialrerun.Boolean != false {
		t.Error(`partialrerun.Boolean != false`)
	}

	needs, ok := pcds["needs"]
	if !ok {
		t.Error(`pcds["needs"] == nil`)
	}

	if needs.Type != PipelineContextDataTypeDictionary {
		t.Error(`needs.Type != PipelineContextDataTypeDictionary`)
	}

	if len(needs.Dictionary) != 0 {
		t.Error(`len(needs.Dictionary) != 0`)
	}

	vars, ok := pcds["vars"]
	if !ok {
		t.Error(`pcds["vars"] == nil`)
	}

	if vars.Type != PipelineContextDataTypeString {
		t.Error(`vars.Type != PipelineContextDataTypeString`)
	}

	if vars.String != "" {
		t.Error(`vars.String != ""`)
	}

	inputs, ok := pcds["inputs"]
	if !ok {
		t.Error(`pcds["inputs"] == nil`)
	}

	if inputs.Type != PipelineContextDataTypeDictionary {
		t.Error(`inputs.Type != PipelineContextDataTypeDictionary`)
	}

	if len(inputs.Dictionary) != 0 {
		t.Error(`len(inputs.Dictionary) != 0`)
	}

	matrix, ok := pcds["matrix"]
	if !ok {
		t.Error(`pcds["matrix"] == nil`)
	}

	if matrix.Type != PipelineContextDataTypeString {
		t.Error(`matrix.Type != PipelineContextDataTypeString`)
	}

	if matrix.String != "" {
		t.Error(`matrix.String != ""`)
	}

	strategy, ok := pcds["strategy"]
	if !ok {
		t.Error(`pcds["strategy"] == nil`)
	}

	if strategy.Type != PipelineContextDataTypeDictionary {
		t.Error(`strategy.Type != PipelineContextDataTypeDictionary`)
	}

	if len(strategy.Dictionary) != 3 {
		t.Error(`len(strategy.Dictionary) != 3`)
	}

	if strategy.Dictionary[0].Key != "fail-fast" {
		t.Error(`strategy.Dictionary[0].Key != "fail-fast"`)
	}

	if strategy.Dictionary[0].Val.Type != PipelineContextDataTypeBoolean {
		t.Error(`strategy.Dictionary[0].Val.Type != PipelineContextDataTypeBoolean`)
	}

	if strategy.Dictionary[0].Val.Boolean != true {
		t.Error(`strategy.Dictionary[0].Val.Boolean != true`)
	}

	if strategy.Dictionary[1].Key != "job-index" {
		t.Error(`strategy.Dictionary[1].Key != "job-index"`)
	}

	if strategy.Dictionary[1].Val.Type != PipelineContextDataTypeNumber {
		t.Error(`strategy.Dictionary[1].Val.Type != PipelineContextDataTypeNumber`)
	}

	if strategy.Dictionary[1].Val.Number != .0 {
		t.Error(`strategy.Dictionary[1].Val.Number != .0`)
	}

	if strategy.Dictionary[2].Key != "job-total" {
		t.Error(`strategy.Dictionary[2].Key != "job-total"`)
	}

	if strategy.Dictionary[2].Val.Type != PipelineContextDataTypeNumber {
		t.Error(`strategy.Dictionary[2].Val.Type != PipelineContextDataTypeNumber`)
	}

	if strategy.Dictionary[2].Val.Number != 1. {
		t.Error(`strategy.Dictionary[2].Val.Number != 1.`)
	}
}
