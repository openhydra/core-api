package privileges

import "fmt"

// IPrivilegeProvider is an interface for checking user privileges
type DefaultPrivilegeProvider struct {
}

// SetFullAccess returns a map of all modules with full access
func (p *DefaultPrivilegeProvider) SetFullAccess() map[string]uint64 {
	result := map[string]uint64{}
	for moduleName, mPermission := range Modules {
		for _, per := range mPermission {
			result[moduleName] |= per
		}
	}
	return result
}

// CanAccess checks if a user has access to a module
func (p *DefaultPrivilegeProvider) CanAccess(permission map[string]uint64, moduleName string, moduleRequiredPermission uint64) (bool, error) {

	if _, found := permission[moduleName]; !found {
		return false, fmt.Errorf("module %s not found", moduleName)
	}

	if permission[moduleName]&moduleRequiredPermission == moduleRequiredPermission {
		return true, nil
	}
	return false, nil
}

// ModulePermission returns a map of all modules with their permissions
func (p *DefaultPrivilegeProvider) ModulePermission(permission map[string]uint64, moduleName string) (map[string]bool, error) {

	if _, found := permission[moduleName]; !found {
		return nil, fmt.Errorf("module %s not found", moduleName)
	}

	var result = map[string]bool{}

	permissionsHave := permission[moduleName]

	for name, per := range Modules[moduleName] {
		if permissionsHave&per == per {
			result[name] = true
		} else {
			result[name] = false
		}
	}

	return result, nil
}
