package wire

import (
	"errors"

	"github.com/microsoft/azure-devops-go-api/azuredevops"

	"github.com/inloco/kube-actions/operator/internal/controller/actionsrunner/util"
)

func IsUnrecoverable(err error) bool {
	return isErrOAuth2InvalidClient(err) || isTaskAgentNotFoundException(err)
}

func isErrOAuth2InvalidClient(err error) bool {
	return errors.Is(err, util.ErrOAuth2InvalidClient)
}

func isTaskAgentNotFoundException(err error) bool {
	wrappedError, ok := err.(azuredevops.WrappedError)
	if !ok {
		return false
	}

	typeKey := wrappedError.TypeKey
	return typeKey != nil && *typeKey == "TaskAgentNotFoundException"
}
