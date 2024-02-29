package calendly

type User struct {
	ID        string `json:"uri"`
	Email     string `json:"email"`
	FullName  string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
}

type OrgMembership struct {
	Org  string `json:"organization"`
	Role string `json:"role"`
	User *User  `json:"user"`
}

type Organization struct {
	ID        string `json:"uri"`
	CreatedAt string `json:"created_at"`
	Plan      string `json:"plan"`
	Stage     string `json:"stage"`
}
