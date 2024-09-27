package v1

import "time"

type SystemInfo struct {
	Company       string    `json:"company,omitempty" description:"do not input auto generator, license company info"`
	Expired       time.Time `json:"end,omitempty" description:"do not input auto generator, license expired date"`
	CPU           string    `json:"cpu,omitempty" description:"do not input auto generator, license cpu limit info"`
	Node          string    `json:"node,omitempty" description:"do not input auto generator, license node limit info"`
	Product       string    `json:"product,omitempty" description:"do not input auto generator, license production info"`
	Version       string    `json:"version,omitempty" description:"do not input auto generator, license version info"`
	MacAddress    string    `json:"mac_address,omitempty" description:"do not input auto generator, license mac address info"`
	LicenseValid  bool      `json:"licenseValid" description:"do not input auto generator, license is valid"`
	SystemVersion string    `json:"systemVersion" description:"do not input auto generator, license system version info"`
	SystemProduct string    `json:"systemProduct" description:"do not input auto generator, license system product info"`
	ErrorDetail   string    `json:"error,omitempty" description:"do not input auto generator, license check failed root cause"`
	License       string    `json:"license" description:"do not input auto generator, license string"`
	Modules       string    `json:"modules" description:"do not input auto generator, such as-> modules:web-console"`
}
