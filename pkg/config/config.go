package config

type ActionBlock struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name"`
}

type ResourceRoleConfig struct {
	Key         string   `json:"key" yaml:"key" mapstructure:"key"`
	Name        string   `json:"name" yaml:"name" mapstructure:"name"`
	Permissions []string `json:"permissions,omitempty" yaml:"permissions,omitempty" mapstructure:"permissions"`
	Extends     []string `json:"extends,omitempty" yaml:"extends,omitempty" mapstructure:"extends"`
}

type RelationConfig struct {
	Key             string `json:"key" yaml:"key" mapstructure:"key"`
	Name            string `json:"name" yaml:"name" mapstructure:"name"`
	SubjectResource string `json:"subject_resource" yaml:"subject_resource" mapstructure:"subject_resource"`
}

type ResourceConfig struct {
	Key       string                 `json:"key" yaml:"key" mapstructure:"key"`
	Name      string                 `json:"name" yaml:"name" mapstructure:"name"`
	Actions   map[string]ActionBlock `json:"actions" yaml:"actions" mapstructure:"actions"`
	Roles     []ResourceRoleConfig   `json:"roles,omitempty" yaml:"roles,omitempty" mapstructure:"roles"`
	Relations []RelationConfig       `json:"relations,omitempty" yaml:"relations,omitempty" mapstructure:"relations"`
}

type RoleConfig struct {
	Key         string   `json:"key" yaml:"key" mapstructure:"key"`
	Name        string   `json:"name" yaml:"name" mapstructure:"name"`
	Permissions []string `json:"permissions,omitempty" yaml:"permissions,omitempty" mapstructure:"permissions"`
}

type ImplicitGrantConfig struct {
	Resource         string `json:"resource" yaml:"resource" mapstructure:"resource"`
	Role             string `json:"role" yaml:"role" mapstructure:"role"`
	OnResource       string `json:"on_resource" yaml:"on_resource" mapstructure:"on_resource"`
	DerivedRole      string `json:"derived_role" yaml:"derived_role" mapstructure:"derived_role"`
	LinkedByRelation string `json:"linked_by_relation" yaml:"linked_by_relation" mapstructure:"linked_by_relation"`
}

type SchemaConfig struct {
	Mode           string                `json:"mode,omitempty" yaml:"mode,omitempty" mapstructure:"mode"`
	Resources      []ResourceConfig      `json:"resources,omitempty" yaml:"resources,omitempty" mapstructure:"resources"`
	Roles          []RoleConfig          `json:"roles,omitempty" yaml:"roles,omitempty" mapstructure:"roles"`
	ImplicitGrants []ImplicitGrantConfig `json:"implicit_grants,omitempty" yaml:"implicit_grants,omitempty" mapstructure:"implicit_grants"`
}

type UserConfig struct {
	Key       string `json:"key" yaml:"key" mapstructure:"key"`
	Email     string `json:"email,omitempty" yaml:"email,omitempty" mapstructure:"email"`
	FirstName string `json:"first_name,omitempty" yaml:"first_name,omitempty" mapstructure:"first_name"`
	LastName  string `json:"last_name,omitempty" yaml:"last_name,omitempty" mapstructure:"last_name"`
}

type TenantConfig struct {
	Key  string `json:"key" yaml:"key" mapstructure:"key"`
	Name string `json:"name" yaml:"name" mapstructure:"name"`
}

type ResourceInstanceConfig struct {
	Key      string `json:"key" yaml:"key" mapstructure:"key"`
	Resource string `json:"resource" yaml:"resource" mapstructure:"resource"`
	Tenant   string `json:"tenant,omitempty" yaml:"tenant,omitempty" mapstructure:"tenant"`
}

type RelationshipTupleConfig struct {
	Subject  string `json:"subject" yaml:"subject" mapstructure:"subject"`
	Relation string `json:"relation" yaml:"relation" mapstructure:"relation"`
	Object   string `json:"object" yaml:"object" mapstructure:"object"`
}

type RoleAssignmentConfig struct {
	User             string `json:"user" yaml:"user" mapstructure:"user"`
	Role             string `json:"role" yaml:"role" mapstructure:"role"`
	Tenant           string `json:"tenant" yaml:"tenant" mapstructure:"tenant"`
	ResourceInstance string `json:"resource_instance,omitempty" yaml:"resource_instance,omitempty" mapstructure:"resource_instance"`
}

type DataConfig struct {
	Tenants            []TenantConfig            `json:"tenants,omitempty" yaml:"tenants,omitempty" mapstructure:"tenants"`
	Users              []UserConfig              `json:"users,omitempty" yaml:"users,omitempty" mapstructure:"users"`
	ResourceInstances  []ResourceInstanceConfig  `json:"resource_instances,omitempty" yaml:"resource_instances,omitempty" mapstructure:"resource_instances"`
	RelationshipTuples []RelationshipTupleConfig `json:"relationship_tuples,omitempty" yaml:"relationship_tuples,omitempty" mapstructure:"relationship_tuples"`
	RoleAssignments    []RoleAssignmentConfig    `json:"role_assignments,omitempty" yaml:"role_assignments,omitempty" mapstructure:"role_assignments"`
}

type Config struct {
	Port   int          `json:"port" yaml:"port" mapstructure:"port"`
	Schema SchemaConfig `json:"schema" yaml:"schema" mapstructure:"schema"`
	Data   DataConfig   `json:"data" yaml:"data" mapstructure:"data"`
}
