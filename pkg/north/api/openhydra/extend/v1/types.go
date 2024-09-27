package v1

import (
	openhydraApi "open-hydra-server-api/pkg/open-hydra/apis"
)

type WrapperSandbox struct {
	Sandboxes          *openhydraApi.PluginList `json:"sandboxes,omitempty"`
	RunningSandboxName string                   `json:"runningSandboxName,omitempty"`
}
