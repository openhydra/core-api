package privileges

import "net/http"

var Modules = map[string]map[string]uint64{
	"course":            CourseFullPermission,
	"dataset":           DatasetFullPermission,
	"device":            DeviceFullPermission,
	"user":              UserFullPermission,
	"role":              RoleFullPermission,
	"rag":               RagFullPermission,
	"setting":           SettingFullPermission,
	"group":             GroupFullPermission,
	"flavor":            FlavorFullPermission,
	"indexView":         IndexPageFullPermission,
	"courseStudentView": CourseStudentFullPermission,
	"deviceStudentView": DeviceStudentFullPermission,
	"model":             ModelFullPermission,
}

func ModulesNoPermission() map[string]uint64 {
	return map[string]uint64{
		"course":            0,
		"dataset":           0,
		"device":            0,
		"user":              0,
		"role":              0,
		"rag":               0,
		"setting":           0,
		"group":             0,
		"flavor":            0,
		"indexView":         0,
		"courseStudentView": 0,
		"deviceStudentView": 0,
		"model":             0,
	}
}

func ModulesFullPermission() map[string]uint64 {
	return map[string]uint64{
		"course":            PermissionCourseCreate | PermissionCourseDelete | PermissionCourseList | PermissionCourseUpdate | PermissionCourseViewPage,
		"dataset":           PermissionDatasetCreate | PermissionDatasetDelete | PermissionDatasetList | PermissionDatasetUpdate | PermissionDatasetViewPage,
		"device":            PermissionDeviceCreate | PermissionDeviceDelete | PermissionDeviceList | PermissionDeviceUpdate | PermissionDeviceViewPage,
		"user":              PermissionUserCreate | PermissionUserDelete | PermissionUserList | PermissionUserUpdate | PermissionUserViewPage,
		"role":              PermissionRoleCreate | PermissionRoleDelete | PermissionRoleList | PermissionRoleUpdate | PermissionRoleViewPage,
		"rag":               PermissionRagCreate | PermissionRagDelete | PermissionRagList | PermissionRagUpdate | PermissionRagViewPage | PermissionRagQuery | PermissionRagManageOtherUserResource,
		"setting":           PermissionSettingCreate | PermissionSettingDelete | PermissionSettingList | PermissionSettingUpdate | PermissionSettingViewPage,
		"group":             PermissionGroupCreate | PermissionGroupDelete | PermissionGroupList | PermissionGroupUpdate | PermissionGroupViewPage,
		"flavor":            PermissionFlavorCreate | PermissionFlavorDelete | PermissionFlavorList | PermissionFlavorUpdate | PermissionFlavorViewPage,
		"indexView":         PermissionIndexPageViewLearningCenter | PermissionIndexPageViewResourceOverall,
		"courseStudentView": PermissionCourseStudentViewPage,
		"deviceStudentView": PermissionDeviceStudentViewPage | PermissionDeviceStudentList | PermissionDeviceStudentCreate | PermissionDeviceStudentUpdate | PermissionDeviceStudentDelete,
		"model":             PermissionModelCreate | PermissionModelDelete | PermissionModelList | PermissionModelUpdate | PermissionModelViewPage,
	}
}

var CourseFullPermission = map[string]uint64{
	"view_page":                  PermissionCourseViewPage,
	"list":                       PermissionCourseList,
	"create":                     PermissionCourseCreate,
	"update":                     PermissionCourseUpdate,
	"delete":                     PermissionCourseDelete,
	"manage_other_user_resource": PermissionCourseManageOtherUserResource,
	http.MethodGet:               PermissionCourseList,
	http.MethodPost:              PermissionCourseCreate,
	http.MethodPut:               PermissionCourseUpdate,
	http.MethodPatch:             PermissionCourseUpdate,
	http.MethodDelete:            PermissionCourseDelete,
}

var DatasetFullPermission = map[string]uint64{
	"view_page":       PermissionDatasetViewPage,
	"list":            PermissionDatasetList,
	"create":          PermissionDatasetCreate,
	"update":          PermissionDatasetUpdate,
	"delete":          PermissionDatasetDelete,
	http.MethodGet:    PermissionDatasetList,
	http.MethodPost:   PermissionDatasetCreate,
	http.MethodPut:    PermissionDatasetUpdate,
	http.MethodPatch:  PermissionDatasetUpdate,
	http.MethodDelete: PermissionDatasetDelete,
}

var DeviceFullPermission = map[string]uint64{
	"view_page":                  PermissionDeviceViewPage,
	"list":                       PermissionDeviceList,
	"create":                     PermissionDeviceCreate,
	"update":                     PermissionDeviceUpdate,
	"delete":                     PermissionDeviceDelete,
	"manage_other_user_resource": PermissionDeviceManageOtherUserResource,
	http.MethodGet:               PermissionDeviceList,
	http.MethodPost:              PermissionDeviceCreate,
	http.MethodPut:               PermissionDeviceUpdate,
	http.MethodPatch:             PermissionDeviceUpdate,
	http.MethodDelete:            PermissionDeviceDelete,
}

var UserFullPermission = map[string]uint64{
	"view_page":                  PermissionUserViewPage,
	"list":                       PermissionUserList,
	"create":                     PermissionUserCreate,
	"update":                     PermissionUserUpdate,
	"delete":                     PermissionUserDelete,
	"manage_other_user_resource": PermissionUserManageOtherUserResource,
	http.MethodGet:               PermissionUserList,
	http.MethodPost:              PermissionUserCreate,
	http.MethodPut:               PermissionUserUpdate,
	http.MethodPatch:             PermissionUserUpdate,
	http.MethodDelete:            PermissionUserDelete,
}

var RoleFullPermission = map[string]uint64{
	"view_page":       PermissionRoleViewPage,
	"list":            PermissionRoleList,
	"create":          PermissionRoleCreate,
	"update":          PermissionRoleUpdate,
	"delete":          PermissionRoleDelete,
	http.MethodGet:    PermissionRoleList,
	http.MethodPost:   PermissionRoleCreate,
	http.MethodPut:    PermissionRoleUpdate,
	http.MethodPatch:  PermissionRoleUpdate,
	http.MethodDelete: PermissionRoleDelete,
}

var RagFullPermission = map[string]uint64{
	"view_page":                  PermissionRagViewPage,
	"list":                       PermissionRagList,
	"create":                     PermissionRagCreate,
	"update":                     PermissionRagUpdate,
	"delete":                     PermissionRagDelete,
	"query":                      PermissionRagQuery,
	"manage_other_user_resource": PermissionRagManageOtherUserResource,
	http.MethodGet:               PermissionRagList,
	http.MethodPost:              PermissionRagCreate,
	http.MethodPut:               PermissionRagUpdate,
	http.MethodPatch:             PermissionRagUpdate,
	http.MethodDelete:            PermissionRagDelete,
}

var SettingFullPermission = map[string]uint64{
	"view_page":       PermissionSettingViewPage,
	"list":            PermissionSettingList,
	"create":          PermissionSettingCreate,
	"update":          PermissionSettingUpdate,
	"delete":          PermissionSettingDelete,
	http.MethodGet:    PermissionSettingList,
	http.MethodPost:   PermissionSettingCreate,
	http.MethodPut:    PermissionSettingUpdate,
	http.MethodPatch:  PermissionSettingUpdate,
	http.MethodDelete: PermissionSettingDelete,
}

var GroupFullPermission = map[string]uint64{
	"view_page":       PermissionGroupViewPage,
	"list":            PermissionGroupList,
	"create":          PermissionGroupCreate,
	"update":          PermissionGroupUpdate,
	"delete":          PermissionGroupDelete,
	http.MethodGet:    PermissionGroupList,
	http.MethodPost:   PermissionGroupCreate,
	http.MethodPut:    PermissionGroupUpdate,
	http.MethodPatch:  PermissionGroupUpdate,
	http.MethodDelete: PermissionGroupDelete,
}

var FlavorFullPermission = map[string]uint64{
	"view_page":       PermissionFlavorViewPage,
	"list":            PermissionFlavorList,
	"create":          PermissionFlavorCreate,
	"update":          PermissionFlavorUpdate,
	"delete":          PermissionFlavorDelete,
	http.MethodGet:    PermissionFlavorList,
	http.MethodPost:   PermissionFlavorCreate,
	http.MethodPut:    PermissionFlavorUpdate,
	http.MethodPatch:  PermissionFlavorUpdate,
	http.MethodDelete: PermissionFlavorDelete,
}

var IndexPageFullPermission = map[string]uint64{
	"view_page_learning_center": PermissionIndexPageViewLearningCenter,
	"view_resource_overall":     PermissionIndexPageViewResourceOverall,
}

var DeviceStudentFullPermission = map[string]uint64{
	"view_page": PermissionDeviceStudentViewPage,
	"list":      PermissionDeviceStudentList,
	"create":    PermissionDeviceStudentCreate,
	"update":    PermissionDeviceStudentUpdate,
	"delete":    PermissionDeviceStudentDelete,
}

var CourseStudentFullPermission = map[string]uint64{
	"view_page": PermissionCourseStudentViewPage,
}

var ModelFullPermission = map[string]uint64{
	"view_page":       PermissionModelViewPage,
	"list":            PermissionModelList,
	"create":          PermissionModelCreate,
	"update":          PermissionModelUpdate,
	"delete":          PermissionModelDelete,
	http.MethodGet:    PermissionModelList,
	http.MethodPost:   PermissionModelCreate,
	http.MethodPut:    PermissionModelUpdate,
	http.MethodPatch:  PermissionModelUpdate,
	http.MethodDelete: PermissionModelDelete,
}

const (
	PermissionCourseViewPage = 1 << iota
	PermissionCourseList
	PermissionCourseCreate
	PermissionCourseUpdate
	PermissionCourseDelete
	PermissionCourseManageOtherUserResource
)

const (
	PermissionDatasetViewPage = 1 << iota
	PermissionDatasetList
	PermissionDatasetCreate
	PermissionDatasetUpdate
	PermissionDatasetDelete
)

const (
	PermissionDeviceViewPage = 1 << iota
	PermissionDeviceList
	PermissionDeviceCreate
	PermissionDeviceUpdate
	PermissionDeviceDelete
	PermissionDeviceManageOtherUserResource
)

const (
	PermissionUserViewPage = 1 << iota
	PermissionUserList
	PermissionUserCreate
	PermissionUserUpdate
	PermissionUserDelete
	PermissionUserManageOtherUserResource
)

const (
	PermissionRoleViewPage = 1 << iota
	PermissionRoleList
	PermissionRoleCreate
	PermissionRoleUpdate
	PermissionRoleDelete
)

const (
	PermissionRagViewPage = 1 << iota
	PermissionRagList
	PermissionRagCreate
	PermissionRagUpdate
	PermissionRagDelete
	PermissionRagQuery
	PermissionRagManageOtherUserResource
)

const (
	PermissionSettingViewPage = 1 << iota
	PermissionSettingList
	PermissionSettingCreate
	PermissionSettingUpdate
	PermissionSettingDelete
)

const (
	PermissionGroupViewPage = 1 << iota
	PermissionGroupList
	PermissionGroupCreate
	PermissionGroupUpdate
	PermissionGroupDelete
)

const (
	PermissionFlavorViewPage = 1 << iota
	PermissionFlavorList
	PermissionFlavorCreate
	PermissionFlavorUpdate
	PermissionFlavorDelete
)

const (
	PermissionIndexPageViewLearningCenter = 1 << iota
	PermissionIndexPageViewResourceOverall
)

const (
	PermissionDeviceStudentViewPage = 1 << iota
	PermissionDeviceStudentList
	PermissionDeviceStudentCreate
	PermissionDeviceStudentUpdate
	PermissionDeviceStudentDelete
)

const (
	PermissionCourseStudentViewPage = 1 << iota
)

const (
	PermissionModelViewPage = 1 << iota
	PermissionModelList
	PermissionModelCreate
	PermissionModelUpdate
	PermissionModelDelete
)

const (
	PermissionNotRequired = 0
)
