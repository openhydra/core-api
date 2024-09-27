package south

import (
	"bytes"
	"core-api/cmd/core-api-server/app/config"
	"core-api/pkg/core/privileges"
	customErr "core-api/pkg/util/error"
	httpHelper "core-api/pkg/util/http"
	"encoding/json"
	"io"
	"net/http"
	deviceV1 "open-hydra-server-api/pkg/apis/open-hydra-api/device/core/v1"

	"github.com/go-chi/chi/v5"
)

// these handlers most like are forwards the request to the openhydra api server

type OpenhydraSouthAPIHandler struct {
	config *config.Config
}

func NewOpenhydraSouthAPIHandler(config *config.Config) *OpenhydraSouthAPIHandler {
	return &OpenhydraSouthAPIHandler{
		config: config,
	}
}

// device apis

func (h *OpenhydraSouthAPIHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devicesList, err := RequestKubeApiserverWithServiceAccount(h.config, DevicesGVR, "", nil, http.MethodGet, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get devices", http.StatusInternalServerError, "FailedToGetDevices", err)
		return
	}

	groupParams := r.URL.Query()
	groups := groupParams["group"]
	if len(groups) > 0 {
		rawDevicesList := &deviceV1.DeviceList{}
		err := json.Unmarshal(devicesList, rawDevicesList)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal devices", http.StatusInternalServerError, "FailedToUnmarshalDevices", err)
			return
		}

		groupProvider, err := initOrGetGroupProvider(h.config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get group provider", http.StatusInternalServerError, "", err)
			return
		}

		result := &deviceV1.DeviceList{}
		result.Kind = "List"
		result.APIVersion = "v1"
		flatGroups := map[string]struct{}{}
		flatGroupUsers := map[string]struct{}{}
		for _, group := range groups {
			flatGroups[group] = struct{}{}
			groupUsers, err := groupProvider.GetGroupUsers(group, nil)
			if err != nil {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get group users", http.StatusInternalServerError, "", err)
				return
			}
			for _, user := range groupUsers {
				flatGroupUsers[user.Name] = struct{}{}
			}
		}

		for _, device := range rawDevicesList.Items {
			if _, ok := flatGroupUsers[device.Spec.ChineseName]; ok {
				result.Items = append(result.Items, device)
			}
		}

		entityToWrite, err := json.Marshal(result)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal devices", http.StatusInternalServerError, "FailedToMarshalDevices", err)
			return
		}
		w.Write(entityToWrite)
	} else {
		w.Write(devicesList)
	}
}

func (h *OpenhydraSouthAPIHandler) GetDevice(w http.ResponseWriter, r *http.Request) {

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "userId is required", http.StatusBadRequest, "UserIdIsRequired", nil)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {

		userProvider, err := initOrGetUserProvider(h.config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get user provider", http.StatusInternalServerError, "", err)
			return
		}

		user, err := userProvider.SearchUserByName(userId, nil)
		if err != nil {
			if _, ok := err.(*customErr.NotFound); !ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusNotFound, "", err)
			} else {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusInternalServerError, "", err)
			}
			return
		}

		_, canAccess, err := SouthAuthorizationControlWithUser(r, w, "device", privileges.PermissionDeviceManageOtherUserResource, user.Id, "device")
		if err != nil {
			return
		}
		if !canAccess {
			return
		}
	}

	device, err := RequestKubeApiserverWithServiceAccount(h.config, DevicesGVR, userId, nil, http.MethodGet, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get device with no auth", http.StatusInternalServerError, "", err)
		return
	}
	w.Write(device)
}

func (h *OpenhydraSouthAPIHandler) CreateDevice(w http.ResponseWriter, r *http.Request) {

	devicePost := &deviceV1.Device{}
	if !h.config.CoreApiConfig.DisableAuth {

		bodyPost, err := io.ReadAll(r.Body)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to read body", http.StatusInternalServerError, "FailedToReadBody", err)
			return
		}

		err = json.Unmarshal(bodyPost, devicePost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to unmarshal body", http.StatusInternalServerError, "FailedToUnmarshalBody", err)
			return
		}

		userProvider, err := initOrGetUserProvider(h.config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get user provider", http.StatusInternalServerError, "", err)
			return
		}

		user, err := userProvider.SearchUserByName(devicePost.Spec.OpenHydraUsername, nil)
		if err != nil {
			if _, ok := err.(*customErr.NotFound); !ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusNotFound, "", err)
			} else {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusInternalServerError, "", err)
			}
			return
		}

		_, canAccess, err := SouthAuthorizationControlWithUser(r, w, "device", privileges.PermissionDeviceManageOtherUserResource, user.Id, "device")
		if err != nil || !canAccess {
			return
		}
	} else {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(devicePost)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to decode device", http.StatusInternalServerError, "FailedToDecodeDevice", err)
			return
		}
	}

	if devicePost.Spec.OpenHydraUsername == "" {
		httpHelper.WriteCustomErrorAndLog(w, "OpenHydraUsername is required", http.StatusBadRequest, "OpenHydraUsernameIsRequired", nil)
		return
	}

	bodyToPost, err := json.Marshal(devicePost)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to marshal device", http.StatusInternalServerError, "FailedToMarshalDevice", err)
		return
	}

	device, err := RequestKubeApiserverWithServiceAccount(h.config, DevicesGVR, "", io.NopCloser(bytes.NewReader(bodyToPost)), http.MethodPost, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create device", http.StatusInternalServerError, "", err)
		return
	}
	w.Write(device)
}

func (h *OpenhydraSouthAPIHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		httpHelper.WriteCustomErrorAndLog(w, "userId is required", http.StatusBadRequest, "UserIdIsRequired", nil)
		return
	}

	if !h.config.CoreApiConfig.DisableAuth {

		userProvider, err := initOrGetUserProvider(h.config)
		if err != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get user provider", http.StatusInternalServerError, "", err)
			return
		}

		user, err := userProvider.SearchUserByName(userId, nil)
		if err != nil {
			if _, ok := err.(*customErr.NotFound); !ok {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusNotFound, "", err)
			} else {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to get user", http.StatusInternalServerError, "", err)
			}
			return
		}

		_, canAccess, err := SouthAuthorizationControlWithUser(r, w, "device", privileges.PermissionDeviceManageOtherUserResource, user.Id, "device")
		if err != nil || !canAccess {
			return
		}
	}

	device, err := RequestKubeApiserverWithServiceAccount(h.config, DevicesGVR, userId, nil, http.MethodDelete, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete device", http.StatusInternalServerError, "", err)
		return
	}
	w.Write(device)
}

// course apis

func (h *OpenhydraSouthAPIHandler) GetCourses(w http.ResponseWriter, r *http.Request) {

	coursesList, err := RequestKubeApiserverWithServiceAccount(h.config, CourseGVR, "", nil, http.MethodGet, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get courses", http.StatusInternalServerError, "FailedToGetCourses", err)
		return
	}
	w.Write(coursesList)
}

func (h *OpenhydraSouthAPIHandler) GetCourse(w http.ResponseWriter, r *http.Request) {

	course, err := RequestKubeApiserverWithServiceAccount(h.config, CourseGVR, chi.URLParam(r, "courseId"), nil, http.MethodGet, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get course", http.StatusInternalServerError, "FailedToGetCourse", err)
		return
	}
	w.Write(course)
}

func (h *OpenhydraSouthAPIHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	// Todo: we should check permisson in future
	// now it's ok to let it play
	// we do not parse the body handle multi-part form data which might slow down the process
	course, err := RequestKubeApiserverWithServiceAccount(h.config, CourseGVR, "", r.Body, http.MethodPost, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create course", http.StatusInternalServerError, "FailedToCreateCourse", err)
		return
	}
	w.Write(course)
}

func (h *OpenhydraSouthAPIHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {

	// if !h.config.CoreApiConfig.DisableAuth {
	// 	_, _, _, err := SouthAuthorizationControlWithModelTReadBody(r, w, "course", privileges.PermissionCourseManageOtherUserResource, "course", func(course *courseV1.Course) string {
	// 		return course.Spec.CreatedBy
	// 	}, r.Body)
	// 	if err != nil {
	// 		return
	// 	}
	// }

	course, err := RequestKubeApiserverWithServiceAccount(h.config, CourseGVR, chi.URLParam(r, "courseId"), nil, http.MethodDelete, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete course", http.StatusInternalServerError, "FailedToDeleteCourse", err)
		return
	}
	w.Write(course)
}

// dataset apis

func (h *OpenhydraSouthAPIHandler) GetDatasets(w http.ResponseWriter, r *http.Request) {

	datasetsList, err := RequestKubeApiserverWithServiceAccount(h.config, DatasetGVR, "", nil, http.MethodGet, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get datasets", http.StatusInternalServerError, "FailedToGetDatasets", err)
		return
	}
	w.Write(datasetsList)
}

func (h *OpenhydraSouthAPIHandler) GetDataset(w http.ResponseWriter, r *http.Request) {

	dataset, err := RequestKubeApiserverWithServiceAccount(h.config, DatasetGVR, chi.URLParam(r, "datasetId"), nil, http.MethodGet, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get dataset", http.StatusInternalServerError, "FailedToGetDataset", err)
		return
	}
	w.Write(dataset)
}

func (h *OpenhydraSouthAPIHandler) CreateDataset(w http.ResponseWriter, r *http.Request) {

	dataset, err := RequestKubeApiserverWithServiceAccount(h.config, DatasetGVR, "", r.Body, http.MethodPost, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to create dataset", http.StatusInternalServerError, "FailedToCreateDataset", err)
		return
	}
	w.Write(dataset)
}

func (h *OpenhydraSouthAPIHandler) DeleteDataset(w http.ResponseWriter, r *http.Request) {

	dataset, err := RequestKubeApiserverWithServiceAccount(h.config, DatasetGVR, chi.URLParam(r, "datasetId"), nil, http.MethodDelete, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to delete dataset", http.StatusInternalServerError, "FailedToDeleteDataset", err)
		return
	}
	w.Write(dataset)
}

// sumup apis
func (h *OpenhydraSouthAPIHandler) GetSumups(w http.ResponseWriter, r *http.Request) {

	sumupsList, err := RequestKubeApiserverWithServiceAccount(h.config, SumupGVR, "", nil, http.MethodGet, r.Header)
	if err != nil {
		httpHelper.WriteCustomErrorAndLog(w, "Failed to get sumups", http.StatusInternalServerError, "FailedToGetSumups", err)
		return
	}
	w.Write(sumupsList)
}
