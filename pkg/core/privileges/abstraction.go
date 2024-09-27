package privileges

type IPrivilegeProvider interface {
	SetFullAccess() map[string]uint64
	CanAccess(permission map[string]uint64, moduleName string, moduleRequiredPermission uint64) (bool, error)
	ModulePermission(permission map[string]uint64, moduleName string) (map[string]bool, error)
}
