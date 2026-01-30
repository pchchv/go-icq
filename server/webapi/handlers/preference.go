package handlers

// PermitDenyData contains permit/deny list information.
type PermitDenyData struct {
	PDMode     int      `json:"pdMode" xml:"pdMode"`
	PermitList []string `json:"permitList,omitempty" xml:"permitList>user,omitempty"`
	DenyList   []string `json:"denyList,omitempty" xml:"denyList>user,omitempty"`
}
