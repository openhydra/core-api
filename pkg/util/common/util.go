package common

import (
	"bufio"
	"bytes"
	"context"
	coreApiLog "core-api/pkg/logger"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
)

func GetStringValueOrDefault(targetValue, defaultValue string) string {
	if targetValue != "" {
		return targetValue
	}
	return defaultValue
}

func RequestStream(requestUrl, httpMethod string, body io.Reader) (*http.Response, error) {
	client := &http.Client{}
	request, err := http.NewRequest(httpMethod, requestUrl, body)
	if err != nil {
		return nil, err
	}

	// set up SSE headers
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Connection", "keep-alive")

	return client.Do(request)
}

func CommonRequestToChannel(requestUrl, httpMethod string, body io.Reader, responseChannel chan string, errorChannel, doneChannel chan struct{}) {
	response, err := RequestStream(requestUrl, httpMethod, body)
	if err != nil {
		errorChannel <- struct{}{}
		return
	}

	defer response.Body.Close()

	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				// End of stream
				coreApiLog.Logger.Debug("End of stream", "requestUrl", requestUrl, "httpMethod", httpMethod)
				doneChannel <- struct{}{}
				return
			}
			// Log the error and write an error response to the client
			coreApiLog.Logger.Error("error ready response stream", "error", err)
			errorChannel <- struct{}{}
			return
		}

		coreApiLog.Logger.Debug("writing response stream", "data", string(line), "requestUrl", requestUrl, "httpMethod", httpMethod)
		responseChannel <- string(line)
	}

}

func CommonStreamRequestRedirect(requestUrl, httpMethod string, expectedResponseCode int, body io.Reader, w http.ResponseWriter) error {
	response, err := RequestStream(requestUrl, httpMethod, body)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != expectedResponseCode {
		// read response body
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("got unexpected response code: %d expected code is: %d with error: '%s'", response.StatusCode, expectedResponseCode, string(body))
	}

	// set up SSE headers for the response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				// End of stream
				coreApiLog.Logger.Debug("End of stream", "requestUrl", requestUrl, "httpMethod", httpMethod)
				break
			}
			// Log the error and write an error response to the client
			coreApiLog.Logger.Error("error ready response stream", "error", err)
			return err
		}

		coreApiLog.Logger.Debug("writing response stream", "data", string(line), "requestUrl", requestUrl, "httpMethod", httpMethod)
		if _, err := w.Write(line); err != nil {
			coreApiLog.Logger.Error("error writing response stream", "error", err)
			return err
		}

		w.(http.Flusher).Flush()
	}

	return nil
}

func CommonRequest(requestUrl, httpMethod, nameServer string, postBody json.RawMessage, header map[string][]string, skipTlsCheck, disableKeepAlive bool, timeout time.Duration) ([]byte, http.Header, int, error) {
	return CommonRequestForwardBody(requestUrl, httpMethod, nameServer, bytes.NewReader(postBody), header, skipTlsCheck, disableKeepAlive, timeout)
}

func CommonRequestForwardBody(requestUrl, httpMethod, nameServer string, postBody io.Reader, header map[string][]string, skipTlsCheck, disableKeepAlive bool, timeout time.Duration) ([]byte, http.Header, int, error) {
	var req *http.Request
	var reqErr error

	req, reqErr = http.NewRequest(httpMethod, requestUrl, postBody)
	if reqErr != nil {
		return []byte{}, nil, http.StatusInternalServerError, reqErr
	}

	for key, val := range header {
		req.Header.Set(key, strings.Join(val, ","))
	}
	client := &http.Client{
		Timeout: 1000 * time.Second,
	}
	// client := &http.Client{
	// 	Timeout: timeout,
	// }
	tr := &http.Transport{
		DisableKeepAlives: disableKeepAlive,
	}
	if skipTlsCheck {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if nameServer != "" {
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 1 * time.Second}
				return d.DialContext(ctx, "udp", nameServer)
			},
		}
		tr.DialContext = (&net.Dialer{
			Resolver: r,
		}).DialContext
	}
	client.Transport = tr
	resp, respErr := client.Do(req)
	if respErr != nil {
		return []byte{}, nil, http.StatusInternalServerError, respErr
	}
	defer resp.Body.Close()
	body, readBodyErr := io.ReadAll(resp.Body)
	if readBodyErr != nil {
		return []byte{}, nil, http.StatusInternalServerError, readBodyErr
	}
	return body, resp.Header, resp.StatusCode, nil
}

func StartMockServer(port int, handlerLoader func(*restful.WebService), stopChan chan struct{}) error {

	svcContainer := restful.NewContainer()
	ws2 := new(restful.WebService)
	if handlerLoader != nil {
		handlerLoader(ws2)
	}

	svcContainer.Add(ws2)

	httpServer := http.Server{
		Handler: svcContainer,
	}

	httpServer.Addr = ":" + strconv.Itoa(port)
	err := httpServer.ListenAndServe()
	if err != nil {
		return err
	}
	<-stopChan
	httpServer.Close()
	return nil
}

func StartMockHttpsServer(port int, handlerLoader func(*restful.WebService), stopChan chan struct{}) error {
	svcContainer := restful.NewContainer()
	ws := new(restful.WebService)
	if handlerLoader != nil {
		handlerLoader(ws)
	}
	svcContainer.Add(ws)

	// Generate a self-signed certificate
	cert, err := generateSelfSignedCert()
	if err != nil {
		return err
	}

	httpServer := http.Server{
		Addr:      ":" + strconv.Itoa(port),
		Handler:   svcContainer,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}

	// Start the server using TLS
	go func() {
		if err := httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			// Handle error
			return
		}
	}()

	// Wait for stop signal
	<-stopChan
	httpServer.Close()
	return nil
}

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Mock Server Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}, nil
}

func CreateDirIfNotExists(dirLocation string) error {
	if _, err := os.Stat(dirLocation); os.IsNotExist(err) {
		return os.MkdirAll(dirLocation, os.ModeDir|0755)
	}
	return nil
}

func BuildPath(keystoneEndpoint, path string) string {
	baseURL, _ := url.Parse(keystoneEndpoint)
	endpoint, _ := url.Parse(path)
	return baseURL.ResolveReference(endpoint).String()
}

// base64Decode decodes a base64 encoded string
func Base64Decode(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

// check is valid ipv4 address
func IsValidIPv4(ip string) bool {
	return net.ParseIP(ip) != nil && net.ParseIP(ip).To4() != nil
}

// check file exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// check dir exists
func DirExists(dirPath string) bool {
	stat, err := os.Stat(dirPath)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

func SplitTrimFilter(s string, sep string) []string {
	splitList := strings.Split(s, sep)
	strList := []string{}
	for _, value := range splitList {
		value = strings.TrimSpace(value)
		if len(value) > 0 {
			strList = append(strList, value)
		}
	}
	return strList
}
