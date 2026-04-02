package store

import "time"

// Common fields for all "read" responses
type BaseFields struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ProjectID      string    `json:"project_id"`
	EnvironmentID  string    `json:"environment_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// --- Schema types ---

type ActionBlock struct {
	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	ID          string  `json:"id,omitempty"`
}

type Resource struct {
	BaseFields
	Key         string                 `json:"key"`
	Name        string                 `json:"name"`
	Urn         *string                `json:"urn,omitempty"`
	Description *string                `json:"description,omitempty"`
	Actions     map[string]ActionBlock `json:"actions,omitempty"`
}

type Role struct {
	BaseFields
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Permissions []string          `json:"permissions,omitempty"`
	Extends     []string          `json:"extends,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

type ResourceRole struct {
	BaseFields
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Extends     []string `json:"extends,omitempty"`
	ResourceID  string   `json:"resource_id"`
}

type Relation struct {
	BaseFields
	Key               string  `json:"key"`
	Name              string  `json:"name"`
	Description       *string `json:"description,omitempty"`
	SubjectResource   string  `json:"subject_resource"`
	SubjectResourceID string  `json:"subject_resource_id"`
	ObjectResourceID  string  `json:"object_resource_id"`
	ObjectResource    string  `json:"object_resource"`
}

type ImplicitGrant struct {
	RoleID           string `json:"role_id"`
	ResourceID       string `json:"resource_id"`
	RelationID       string `json:"relation_id"`
	Role             string `json:"role"`               // source role key (user must have this)
	OnResource       string `json:"on_resource"`        // source resource type
	LinkedByRelation string `json:"linked_by_relation"` // relation to follow
	TargetResource   string `json:"-"`                  // target resource type (from URL path)
	TargetRole       string `json:"-"`                  // target role key (from URL path)
}

// --- Facts types ---

type Tenant struct {
	BaseFields
	Key          string                 `json:"key"`
	Name         string                 `json:"name"`
	Description  *string                `json:"description,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	LastActionAt time.Time              `json:"last_action_at"`
}

type User struct {
	BaseFields
	Key        string                 `json:"key"`
	Email      *string                `json:"email,omitempty"`
	FirstName  *string                `json:"first_name,omitempty"`
	LastName   *string                `json:"last_name,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type ResourceInstance struct {
	BaseFields
	Key        string                 `json:"key"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id"`
	Tenant     *string                `json:"tenant,omitempty"`
	TenantID   *string                `json:"tenant_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type RelationshipTuple struct {
	BaseFields
	Subject    string `json:"subject"`
	Relation   string `json:"relation"`
	Object     string `json:"object"`
	Tenant     string `json:"tenant"`
	SubjectID  string `json:"subject_id"`
	RelationID string `json:"relation_id"`
	ObjectID   string `json:"object_id"`
	TenantID   string `json:"tenant_id"`
}

type RoleAssignment struct {
	ID               string    `json:"id"`
	User             string    `json:"user"`
	Role             string    `json:"role"`
	Tenant           string    `json:"tenant"`
	ResourceInstance *string   `json:"resource_instance,omitempty"`
	UserID           string    `json:"user_id"`
	RoleID           string    `json:"role_id"`
	TenantID         string    `json:"tenant_id"`
	OrganizationID   string    `json:"organization_id"`
	ProjectID        string    `json:"project_id"`
	EnvironmentID    string    `json:"environment_id"`
	CreatedAt        time.Time `json:"created_at"`
}
