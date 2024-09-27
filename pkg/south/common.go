package south

import (
	"core-api/cmd/core-api-server/app/config"
	auth "core-api/pkg/core/auth"
	"core-api/pkg/core/privileges"
	coreUserV1 "core-api/pkg/north/api/user/core/v1"
	httpHelper "core-api/pkg/util/http"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

var userProvider auth.IUserProvider
var groupProvider auth.IGroupProvider

var (
	OpenHydraApiPrefix = "apis"
	DevicesGVR         = schema.GroupVersionResource{Version: "v1", Resource: "devices", Group: "open-hydra-server.openhydra.io"}
	CourseGVR          = schema.GroupVersionResource{Version: "v1", Resource: "courses", Group: "open-hydra-server.openhydra.io"}
	DatasetGVR         = schema.GroupVersionResource{Version: "v1", Resource: "datasets", Group: "open-hydra-server.openhydra.io"}
	SumupGVR           = schema.GroupVersionResource{Version: "v1", Resource: "sumups", Group: "open-hydra-server.openhydra.io"}
)

func initOrGetUserProvider(serverConfig *config.Config) (auth.IUserProvider, error) {
	if userProvider == nil {
		userProvider, _ = auth.CreateUserProvider(serverConfig, auth.KeystoneAuthProvider)
	}
	return userProvider, nil
}

func initOrGetGroupProvider(serverConfig *config.Config) (auth.IGroupProvider, error) {
	if groupProvider == nil {
		groupProvider, _ = auth.CreateGroupProvider(serverConfig, auth.KeystoneAuthProvider)
	}
	return groupProvider, nil
}

func RequestKubeApiserverWithServiceAccount(config *config.Config, resourcePath schema.GroupVersionResource, extraPath string, postBody io.ReadCloser, method string, header http.Header) ([]byte, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(config.KubeConfig.RestConfig.CAData))
	//caCertPool.AppendCertsFromPEM([]byte(caCert))
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	url := path.Join(OpenHydraApiPrefix, resourcePath.Group, resourcePath.Version, resourcePath.Resource)
	if extraPath != "" {
		url = path.Join(url, extraPath)
	}

	url = fmt.Sprintf("%s/%s", config.KubeConfig.RestConfig.Host, url)

	req, err := http.NewRequest(method, url, postBody)
	if err != nil {
		return nil, err
	}

	if header != nil {
		req.Header = header
	}

	// remove core-api-server's token late when we are about to send the request with in-cluster token
	req.Header.Del("Authorization")
	// note there is a newline character at the end of the token
	// we need to remove it
	req.Header.Add("Authorization", "Bearer "+strings.TrimRight(config.KubeConfig.RestConfig.BearerToken, "\n"))

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return []byte{}, nil
	}

	// read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("request failed with status code %d with error %s", resp.StatusCode, string(body))
	}

	return body, nil
}
func RequestKubeApiserverWithServiceAccountAndParseToT[T any](config *config.Config, resourcePath schema.GroupVersionResource, extraPath string, postBody io.ReadCloser, method string, header http.Header) (*T, error) {

	body, err := RequestKubeApiserverWithServiceAccount(config, resourcePath, extraPath, postBody, method, header)
	if err != nil {
		return nil, err
	}

	result := new(T)

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func ForwardResponseHeader(w http.ResponseWriter, headers http.Header) {
	for key, header := range headers {
		w.Header().Set(key, strings.Join(header, ","))
	}
}

func SouthAuthorizationControlWithModelTReadBody[T any](r *http.Request, w http.ResponseWriter, moduleName string, permissionRequired uint64, resourceName string, idToCompareHandler func(*T) string, body io.ReadCloser) (*coreUserV1.CoreUser, *T, bool, error) {

	if idToCompareHandler == nil {
		return nil, nil, false, fmt.Errorf("idToCompareHandler is nil")
	}

	result := new(T)
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, false, err
	}

	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, nil, false, err
	}

	user, canAccess, err := SouthAuthorizationControlWithUser(r, w, moduleName, permissionRequired, idToCompareHandler(result), resourceName)
	return user, result, canAccess, err
}

func SouthAuthorizationControlWithUser(r *http.Request, w http.ResponseWriter, moduleName string, permissionRequired uint64, userIdToCompareLogin, resourceName string) (*coreUserV1.CoreUser, bool, error) {
	userInMiddle, ok := r.Context().Value("core-user").(*coreUserV1.CoreUser)
	if !ok {
		if w != nil {
			httpHelper.WriteCustomErrorAndLog(w, "Failed to get user from context", http.StatusInternalServerError, "", fmt.Errorf("failed to get user from context"))
		}
		return nil, false, fmt.Errorf("failed to get user from context")
	}
	if userInMiddle.Id != userIdToCompareLogin {
		_, canAccess, err := SouthAuthorizationControl(r, moduleName, permissionRequired)
		if err != nil {
			if w != nil {
				httpHelper.WriteCustomErrorAndLog(w, "Failed to check permission", http.StatusInternalServerError, "", err)
			}
			return userInMiddle, false, err
		}
		if !canAccess {
			if w != nil {
				httpHelper.WriteCustomErrorAndLog(w, "user cannot manage resource that do not own", http.StatusForbidden, "", fmt.Errorf("user '%s' cannot manage resource '%s' that do not own required permission: '%s:%d'", userInMiddle.Id, resourceName, moduleName, permissionRequired))
			}
			return userInMiddle, false, nil
		}
	}
	return userInMiddle, true, nil
}

func SouthAuthorizationControl(r *http.Request, moduleName string, permissionRequired uint64) (*coreUserV1.CoreUser, bool, error) {
	priProvider := &privileges.DefaultPrivilegeProvider{}
	userInMiddle, ok := r.Context().Value("core-user").(*coreUserV1.CoreUser)
	if !ok {
		return nil, false, fmt.Errorf("failed to get user from context")
	}
	canAccess, err := priProvider.CanAccess(userInMiddle.Permission, moduleName, permissionRequired)
	return userInMiddle, canAccess, err
}

func GetFileChatFileName(ragFilePath, kbId string) (string, error) {
	destinationDir := filepath.Join(ragFilePath, "data", "temp", kbId)

	fileName := "un-know"

	entries, err := os.ReadDir(destinationDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileName = entry.Name()
		break
	}

	if fileName == "un-know" {
		return "", fmt.Errorf("no file found in %s", destinationDir)
	}
	return fileName, nil
}
