package v1

import (
	summaryV1 "open-hydra-server-api/pkg/apis/open-hydra-api/summary/core/v1"
	openhydraApi "open-hydra-server-api/pkg/open-hydra/apis"
)

// flavor is merged from openhydra sumups and openhydra-plugin
type Flavor struct {
	SumUps *summaryV1.SumUp         `json:"sumUps,omitempty"`
	Plugin *openhydraApi.PluginList `json:"plugin,omitempty"`
}
