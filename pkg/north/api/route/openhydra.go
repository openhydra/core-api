package route

import (
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	"core-api/pkg/south"
	"fmt"
	"net/http"
)

const (
	openhydraPrefix     = "/apis"
	openhydraAPIVersion = "v1"
	openhydraGroup      = "open-hydra-server.openhydra.io"
)

var openhydraFullPathPrefix = fmt.Sprintf("%s/%s/%s", openhydraPrefix, openhydraGroup, openhydraAPIVersion)

func GetOpenhydraRoute(config *config.Config) *ChiRouteBuilder {
	// init handler by passing the config
	openhydraHandler := south.NewOpenhydraSouthAPIHandler(config)
	return &ChiRouteBuilder{
		PathPrefix: openhydraFullPathPrefix,
		MethodHandlers: []ChiSubRouteBuilder{
			{
				Method:  http.MethodGet,
				Pattern: "/",
				Handler: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("root of openhydra api v1")) },
				ModuleAndPermission: ModuleAndPermission{
					Module:     "",
					Permission: 0,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/devices",
				Handler: openhydraHandler.GetDevices,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "deviceStudentView",
					Permission: privileges.PermissionDeviceStudentViewPage,
				},
			},
			{
				Method:  http.MethodPost,
				Pattern: "/devices",
				Handler: openhydraHandler.CreateDevice,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "deviceStudentView",
					Permission: privileges.PermissionDeviceStudentCreate,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/devices/{userId}",
				Handler: openhydraHandler.GetDevice,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "deviceStudentView",
					Permission: privileges.PermissionDeviceStudentViewPage,
				},
			},
			// note so for no need to support update device
			// {
			// 	Method:  http.MethodPut,
			// 	Pattern: "/devices/{userId}",
			// 	Handler: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("update device by id")) },
			// 	ModuleAndPermission: ModuleAndPermission{
			// 		Module:     "device",
			// 		Permission: privileges.PermissionDeviceUpdate,
			// 	},
			// },
			{
				Method:  http.MethodDelete,
				Pattern: "/devices/{userId}",
				Handler: openhydraHandler.DeleteDevice,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "deviceStudentView",
					Permission: privileges.PermissionDeviceStudentDelete,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/courses",
				Handler: openhydraHandler.GetCourses,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "course",
					Permission: privileges.PermissionCourseList,
				},
			},
			{
				Method:  http.MethodPost,
				Pattern: "/courses",
				Handler: openhydraHandler.CreateCourse,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "course",
					Permission: privileges.PermissionCourseCreate,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/courses/{courseId}",
				Handler: openhydraHandler.GetCourse,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "course",
					Permission: privileges.PermissionCourseList,
				},
			},
			// {
			// 	Method:  http.MethodPut,
			// 	Pattern: "/courses/{id}",
			// 	Handler: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("update course by id")) },
			// 	ModuleAndPermission: ModuleAndPermission{
			// 		Module:     "course",
			// 		Permission: privileges.PermissionCourseUpdate,
			// 	},
			// },
			{
				Method:  http.MethodDelete,
				Pattern: "/courses/{courseId}",
				Handler: openhydraHandler.DeleteCourse,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "course",
					Permission: privileges.PermissionCourseDelete,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/datasets",
				Handler: openhydraHandler.GetDatasets,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "dataset",
					Permission: privileges.PermissionDatasetList,
				},
			},
			{
				Method:  http.MethodPost,
				Pattern: "/datasets",
				Handler: openhydraHandler.CreateDataset,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "dataset",
					Permission: privileges.PermissionDatasetCreate,
				},
			},
			{
				Method:  http.MethodGet,
				Pattern: "/datasets/{datasetId}",
				Handler: openhydraHandler.GetDataset,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "dataset",
					Permission: privileges.PermissionDatasetList,
				},
			},
			// {
			// 	Method:  http.MethodPut,
			// 	Pattern: "/datasets/{id}",
			// 	Handler: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("update dataset by id")) },
			// 	ModuleAndPermission: ModuleAndPermission{
			// 		Module:     "dataset",
			// 		Permission: privileges.PermissionDatasetUpdate,
			// 	},
			// },
			{
				Method:  http.MethodDelete,
				Pattern: "/datasets/{datasetId}",
				Handler: openhydraHandler.DeleteDataset,
				ModuleAndPermission: ModuleAndPermission{
					Module:     "dataset",
					Permission: privileges.PermissionDatasetDelete,
				},
			},
		}}
}
