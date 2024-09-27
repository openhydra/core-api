package train

const (
	// LoadPermission is a constant of type string
	// indicate function handle user should also load LoadPermission details
	LoadPermission = "loadPermission"
	// loadPasswd is a constant of type string
	// indicate function handle user should also load password
	LoadPasswd = "loadPasswd"
	// LoadUnRelatedKeystoneRoles is a constant of type string
	// indicate whether role handler should skip role that do not have corerole object
	LoadUnRelatedKeystoneObjects = "loadUnRelatedKeystoneObjects"
	// UpdateGroupForAllUser is a constant of type string
	// UpdateGroupForAllUser function handle group delete or update should also update all user's group
	UpdateGroupForAllUser = "loadGroup"
	// ReverseGetGroupUsers is a constant of type string
	// ReverseGetGroupUsers function will force GetGroupUsers to reverse the search result
	ReverseGetGroupUsers = "reverseGetGroupUsers"
)
