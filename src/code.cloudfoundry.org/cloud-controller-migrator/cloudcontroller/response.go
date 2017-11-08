package cloudcontroller

type PaginatedResponse struct {
	NextURL     *string `json:"next_url"`
	PreviousURL *string `json:"prev_url"`
}

type ListOrganizationsResponse struct {
	PaginatedResponse
	Resources []OrganizationResource `json:"resources"`
}

type OrganizationResource struct {
	MetadataResource
	Entity struct {
		Name               string `json:"name"`
		SpacesURL          string `json:"spaces_url"`
		UsersURL           string `json:"users_url"`
		ManagersURL        string `json:"managers_url"`
		BillingManagersURL string `json:"billing_managers_url"`
		AuditorsURL        string `json:"auditors_url"`
	} `json:"entity"`
}

type ListSpacesResponse struct {
	PaginatedResponse
	Resources []SpaceResource `json:"resources"`
}

type SpaceResource struct {
	MetadataResource
	Entity struct {
		Name          string `json:"name"`
		DevelopersURL string `json:"developers_url"`
		AuditorsURL   string `json:"auditors_url"`
		ManagersURL   string `json:"managers_url"`
	} `json:"entity"`
}

type MetadataResource struct {
	Metadata struct {
		GUID string `json:"guid"`
		URL  string `json:"url"`
	} `json:"metadata"`
}

type ListUsersResponse struct {
	PaginatedResponse
	Resources []UserResource `json:"resources"`
}

type UserResource struct {
	MetadataResource
	Entity struct {
		Username string `json:"username"`
	} `json:"entity"`
}