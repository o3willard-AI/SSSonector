package access

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Role represents a user role with associated permissions
type Role struct {
	Name        string
	Permissions []Permission
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Permission represents an individual permission
type Permission struct {
	Name        string
	Description string
	Resource    string
	Action      string
}

// Policy represents an access control policy
type Policy struct {
	Name        string
	Description string
	Roles       []string
	Resources   []string
	Actions     []string
	Conditions  map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RBACManager manages role-based access control
type RBACManager struct {
	roles      map[string]*Role
	policies   map[string]*Policy
	roleLock   sync.RWMutex
	policyLock sync.RWMutex
	logger     *zap.Logger
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(logger *zap.Logger) *RBACManager {
	return &RBACManager{
		roles:    make(map[string]*Role),
		policies: make(map[string]*Policy),
		logger:   logger,
	}
}

// AddRole adds a new role to the RBAC system
func (m *RBACManager) AddRole(role *Role) error {
	m.roleLock.Lock()
	defer m.roleLock.Unlock()

	if _, exists := m.roles[role.Name]; exists {
		return fmt.Errorf("role %s already exists", role.Name)
	}

	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	m.roles[role.Name] = role

	m.logger.Info("Added new role",
		zap.String("role", role.Name),
		zap.Int("permissions", len(role.Permissions)))

	return nil
}

// GetRole retrieves a role by name
func (m *RBACManager) GetRole(name string) (*Role, error) {
	m.roleLock.RLock()
	defer m.roleLock.RUnlock()

	role, exists := m.roles[name]
	if !exists {
		return nil, fmt.Errorf("role %s not found", name)
	}

	return role, nil
}

// UpdateRole updates an existing role
func (m *RBACManager) UpdateRole(role *Role) error {
	m.roleLock.Lock()
	defer m.roleLock.Unlock()

	if _, exists := m.roles[role.Name]; !exists {
		return fmt.Errorf("role %s does not exist", role.Name)
	}

	role.UpdatedAt = time.Now()
	m.roles[role.Name] = role

	m.logger.Info("Updated role",
		zap.String("role", role.Name),
		zap.Int("permissions", len(role.Permissions)))

	return nil
}

// DeleteRole removes a role from the RBAC system
func (m *RBACManager) DeleteRole(name string) error {
	m.roleLock.Lock()
	defer m.roleLock.Unlock()

	if _, exists := m.roles[name]; !exists {
		return fmt.Errorf("role %s does not exist", name)
	}

	delete(m.roles, name)

	m.logger.Info("Deleted role",
		zap.String("role", name))

	return nil
}

// AddPolicy adds a new access control policy
func (m *RBACManager) AddPolicy(policy *Policy) error {
	m.policyLock.Lock()
	defer m.policyLock.Unlock()

	if _, exists := m.policies[policy.Name]; exists {
		return fmt.Errorf("policy %s already exists", policy.Name)
	}

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	m.policies[policy.Name] = policy

	m.logger.Info("Added new policy",
		zap.String("policy", policy.Name),
		zap.Strings("roles", policy.Roles),
		zap.Strings("resources", policy.Resources))

	return nil
}

// GetPolicy retrieves a policy by name
func (m *RBACManager) GetPolicy(name string) (*Policy, error) {
	m.policyLock.RLock()
	defer m.policyLock.RUnlock()

	policy, exists := m.policies[name]
	if !exists {
		return nil, fmt.Errorf("policy %s not found", name)
	}

	return policy, nil
}

// UpdatePolicy updates an existing policy
func (m *RBACManager) UpdatePolicy(policy *Policy) error {
	m.policyLock.Lock()
	defer m.policyLock.Unlock()

	if _, exists := m.policies[policy.Name]; !exists {
		return fmt.Errorf("policy %s does not exist", policy.Name)
	}

	policy.UpdatedAt = time.Now()
	m.policies[policy.Name] = policy

	m.logger.Info("Updated policy",
		zap.String("policy", policy.Name),
		zap.Strings("roles", policy.Roles))

	return nil
}

// DeletePolicy removes a policy from the RBAC system
func (m *RBACManager) DeletePolicy(name string) error {
	m.policyLock.Lock()
	defer m.policyLock.Unlock()

	if _, exists := m.policies[name]; !exists {
		return fmt.Errorf("policy %s does not exist", name)
	}

	delete(m.policies, name)

	m.logger.Info("Deleted policy",
		zap.String("policy", name))

	return nil
}

// CheckPermission checks if a role has a specific permission
func (m *RBACManager) CheckPermission(roleName, resource, action string) (bool, error) {
	m.roleLock.RLock()
	defer m.roleLock.RUnlock()

	role, exists := m.roles[roleName]
	if !exists {
		return false, fmt.Errorf("role %s not found", roleName)
	}

	for _, permission := range role.Permissions {
		if permission.Resource == resource && permission.Action == action {
			return true, nil
		}
	}

	return false, nil
}

// CheckPolicy checks if a request complies with access policies
func (m *RBACManager) CheckPolicy(ctx context.Context, roleName, resource, action string) (bool, error) {
	m.policyLock.RLock()
	defer m.policyLock.RUnlock()

	// First check if the role has direct permission
	hasPermission, err := m.CheckPermission(roleName, resource, action)
	if err != nil {
		return false, err
	}
	if hasPermission {
		return true, nil
	}

	// Check policies that apply to the role
	for _, policy := range m.policies {
		for _, role := range policy.Roles {
			if role == roleName {
				// Check if resource is allowed
				for _, allowedResource := range policy.Resources {
					if allowedResource == resource || allowedResource == "*" {
						// Check if action is allowed
						for _, allowedAction := range policy.Actions {
							if allowedAction == action || allowedAction == "*" {
								return true, nil
							}
						}
					}
				}
			}
		}
	}

	return false, nil
}

// ListRoles returns all defined roles
func (m *RBACManager) ListRoles() ([]*Role, error) {
	m.roleLock.RLock()
	defer m.roleLock.RUnlock()

	roles := make([]*Role, 0, len(m.roles))
	for _, role := range m.roles {
		roles = append(roles, role)
	}

	return roles, nil
}

// ListPolicies returns all defined policies
func (m *RBACManager) ListPolicies() ([]*Policy, error) {
	m.policyLock.RLock()
	defer m.policyLock.RUnlock()

	policies := make([]*Policy, 0, len(m.policies))
	for _, policy := range m.policies {
		policies = append(policies, policy)
	}

	return policies, nil
}

// GetRolePermissions returns all permissions for a specific role
func (m *RBACManager) GetRolePermissions(roleName string) ([]Permission, error) {
	m.roleLock.RLock()
	defer m.roleLock.RUnlock()

	role, exists := m.roles[roleName]
	if !exists {
		return nil, fmt.Errorf("role %s not found", roleName)
	}

	return role.Permissions, nil
}

// AddPermissionToRole adds a permission to an existing role
func (m *RBACManager) AddPermissionToRole(roleName string, permission Permission) error {
	m.roleLock.Lock()
	defer m.roleLock.Unlock()

	role, exists := m.roles[roleName]
	if !exists {
		return fmt.Errorf("role %s not found", roleName)
	}

	// Check if permission already exists
	for _, p := range role.Permissions {
		if p.Resource == permission.Resource && p.Action == permission.Action {
			return fmt.Errorf("permission %s:%s already exists for role %s",
				permission.Resource, permission.Action, roleName)
		}
	}

	role.Permissions = append(role.Permissions, permission)
	role.UpdatedAt = time.Now()

	m.logger.Info("Added permission to role",
		zap.String("role", roleName),
		zap.String("resource", permission.Resource),
		zap.String("action", permission.Action))

	return nil
}

// RemovePermissionFromRole removes a permission from an existing role
func (m *RBACManager) RemovePermissionFromRole(roleName, resource, action string) error {
	m.roleLock.Lock()
	defer m.roleLock.Unlock()

	role, exists := m.roles[roleName]
	if !exists {
		return fmt.Errorf("role %s not found", roleName)
	}

	newPermissions := []Permission{}
	found := false

	for _, p := range role.Permissions {
		if p.Resource == resource && p.Action == action {
			found = true
			continue
		}
		newPermissions = append(newPermissions, p)
	}

	if !found {
		return fmt.Errorf("permission %s:%s not found for role %s",
			resource, action, roleName)
	}

	role.Permissions = newPermissions
	role.UpdatedAt = time.Now()

	m.logger.Info("Removed permission from role",
		zap.String("role", roleName),
		zap.String("resource", resource),
		zap.String("action", action))

	return nil
}

// DefaultRoles returns the default set of roles for SSSonector
func DefaultRoles() []*Role {
	now := time.Now()
	return []*Role{
		{
			Name: "admin",
			Permissions: []Permission{
				{Name: "full_access", Description: "Full system access", Resource: "*", Action: "*"},
				{Name: "manage_users", Description: "Manage user accounts", Resource: "users", Action: "*"},
				{Name: "manage_certificates", Description: "Manage certificates", Resource: "certificates", Action: "*"},
				{Name: "manage_config", Description: "Manage configuration", Resource: "config", Action: "*"},
				{Name: "view_audit", Description: "View audit logs", Resource: "audit", Action: "read"},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Name: "operator",
			Permissions: []Permission{
				{Name: "manage_tunnels", Description: "Manage tunnel connections", Resource: "tunnels", Action: "*"},
				{Name: "view_status", Description: "View system status", Resource: "status", Action: "read"},
				{Name: "view_metrics", Description: "View system metrics", Resource: "metrics", Action: "read"},
				{Name: "restart_services", Description: "Restart services", Resource: "services", Action: "restart"},
				{Name: "view_config", Description: "View configuration", Resource: "config", Action: "read"},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Name: "monitor",
			Permissions: []Permission{
				{Name: "view_status", Description: "View system status", Resource: "status", Action: "read"},
				{Name: "view_metrics", Description: "View system metrics", Resource: "metrics", Action: "read"},
				{Name: "view_logs", Description: "View system logs", Resource: "logs", Action: "read"},
				{Name: "view_audit", Description: "View audit logs", Resource: "audit", Action: "read"},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Name: "client",
			Permissions: []Permission{
				{Name: "connect_tunnel", Description: "Connect to tunnel", Resource: "tunnels", Action: "connect"},
				{Name: "view_status", Description: "View client status", Resource: "status", Action: "read"},
				{Name: "view_metrics", Description: "View client metrics", Resource: "metrics", Action: "read"},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// DefaultPolicies returns the default set of policies for SSSonector
func DefaultPolicies() []*Policy {
	now := time.Now()
	return []*Policy{
		{
			Name:        "tunnel_management",
			Description: "Policy for managing tunnel connections",
			Roles:       []string{"admin", "operator"},
			Resources:   []string{"tunnels", "connections"},
			Actions:     []string{"create", "read", "update", "delete", "restart"},
			Conditions:  map[string]string{},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        "certificate_management",
			Description: "Policy for managing certificates",
			Roles:       []string{"admin"},
			Resources:   []string{"certificates", "keys"},
			Actions:     []string{"create", "read", "update", "delete", "revoke"},
			Conditions:  map[string]string{},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        "monitoring_access",
			Description: "Policy for accessing monitoring data",
			Roles:       []string{"admin", "operator", "monitor"},
			Resources:   []string{"status", "metrics", "logs", "audit"},
			Actions:     []string{"read"},
			Conditions:  map[string]string{},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Name:        "configuration_management",
			Description: "Policy for managing system configuration",
			Roles:       []string{"admin"},
			Resources:   []string{"config"},
			Actions:     []string{"create", "read", "update", "delete"},
			Conditions:  map[string]string{},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}
