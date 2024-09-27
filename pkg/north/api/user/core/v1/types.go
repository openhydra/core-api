package v1

// swagger:response userUpdate
type CoreUser struct {
	Id          string            `json:"id,omitempty"`
	Name        string            `json:"name,omitempty"`
	Email       string            `json:"email,omitempty"`
	Description string            `json:"description,omitempty"`
	Password    string            `json:"password,omitempty"`
	Roles       []CoreRole        `json:"roles,omitempty"`
	Groups      []CoreGroup       `json:"groups,omitempty"`
	Permission  map[string]uint64 `json:"permission,omitempty"`
	UnEditable  bool              `json:"uneditable,omitempty"`
}

// swagger:response roleUpdate
type CoreRole struct {
	Id          string            `json:"id,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Permission  map[string]uint64 `json:"permission,omitempty"`
	UnEditable  bool              `json:"uneditable,omitempty"`
}

type CoreGroup struct {
	Id          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	TagColor    string `json:"tagColor,omitempty"`
}

type CoreGroupSummary struct {
	Counts []CoreGroupSummaryDetail `json:"counts"`
}

type CoreGroupSummaryDetail struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}
