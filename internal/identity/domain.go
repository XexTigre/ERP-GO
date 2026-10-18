package identity

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleCataloger Role = "cataloger"
	RoleStock     Role = "stock"
	RoleReader    Role = "reader"
)

type Resource string

const (
	ResourceAdmin   Resource = "admin"
	ResourceCatalog Resource = "catalog"
	ResourceStock   Resource = "stock"
)

type Action string

const (
	Read  Action = "read"
	Write Action = "write"
)

var writes = map[Role]map[Resource]bool{
	RoleAdmin:     {ResourceAdmin: true, ResourceCatalog: true, ResourceStock: true},
	RoleCataloger: {ResourceCatalog: true}, RoleStock: {ResourceStock: true},
}

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleCataloger || r == RoleStock || r == RoleReader
}

type Principal struct {
	TenantID string
	KeyID    string
	Role     Role
}

type Branch struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (p Principal) Allowed(resource Resource, action Action) bool {
	if !p.Role.Valid() {
		return false
	}
	if resource != ResourceAdmin && resource != ResourceCatalog && resource != ResourceStock {
		return false
	}
	if action == Read {
		return true
	}
	return action == Write && writes[p.Role][resource]
}
func (p Principal) CanWrite(resource Resource) bool { return p.Allowed(resource, Write) }
