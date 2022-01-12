package wire

import (
	"context"
	"fmt"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/itchyny/gojq"
)

type PolicyValidator struct {
	ruleCache map[inlocov1alpha1.ActionsRunnerPolicyRule]*gojq.Code
}

func NewPolicyValidator() *PolicyValidator {
	return &PolicyValidator{
		ruleCache: make(map[inlocov1alpha1.ActionsRunnerPolicyRule]*gojq.Code),
	}
}

func (pv *PolicyValidator) Validate(ctx context.Context, policy *inlocov1alpha1.ActionsRunnerPolicy, pajr *PipelineAgentJobRequest) (*inlocov1alpha1.ActionsRunnerPolicyRule, error) {
	contextData := make(map[string]interface{}, len(pajr.ContextData))
	for k, v := range pajr.ContextData {
		flattened, err := v.Flatten()
		if err != nil {
			return nil, err
		}

		contextData[k] = flattened
	}

	must, err := pv.validateMust(ctx, policy.Must, contextData)
	if err != nil {
		return nil, err
	}
	if must != nil {
		return must, nil
	}

	mustNot, err := pv.validateMustNot(ctx, policy.MustNot, contextData)
	if err != nil {
		return nil, err
	}
	if mustNot != nil {
		return mustNot, nil
	}

	return nil, nil
}

func (pv *PolicyValidator) validateMust(ctx context.Context, rules []inlocov1alpha1.ActionsRunnerPolicyRule, contextData map[string]interface{}) (*inlocov1alpha1.ActionsRunnerPolicyRule, error) {
	for _, rule := range rules {
		code, err := pv.compileRule(rule)
		if err != nil {
			return nil, err
		}

		it := code.RunWithContext(ctx, contextData)

		el, ok := it.Next()
		if !ok {
			return &rule, nil
		}

		b, ok := el.(bool)
		if !ok {
			return &rule, nil
		}
		if !b {
			return &rule, nil
		}
	}

	return nil, nil
}

func (pv *PolicyValidator) validateMustNot(ctx context.Context, rules []inlocov1alpha1.ActionsRunnerPolicyRule, contextData map[string]interface{}) (*inlocov1alpha1.ActionsRunnerPolicyRule, error) {
	for _, rule := range rules {
		code, err := pv.compileRule(rule)
		if err != nil {
			return nil, err
		}

		it := code.RunWithContext(ctx, contextData)

		el, ok := it.Next()
		if !ok {
			continue
		}

		b, ok := el.(bool)
		if !ok {
			continue
		}
		if b {
			return &rule, nil
		}
	}

	return nil, nil
}

func (pv *PolicyValidator) compileRule(rule inlocov1alpha1.ActionsRunnerPolicyRule) (*gojq.Code, error) {
	code, ok := pv.ruleCache[rule]
	if ok {
		return code, nil
	}

	q, err := gojq.Parse(string(rule))
	if err != nil {
		return nil, fmt.Errorf("unable to parse policy rule `%s`: %w", rule, err)
	}

	c, err := gojq.Compile(q)
	if err != nil {
		return nil, fmt.Errorf("unable to compile policy rule `%s`: %w", rule, err)
	}

	pv.ruleCache[rule] = c
	return c, nil
}
