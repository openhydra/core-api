package v1

type NvidiaGpuShare struct {
	Version string                `json:"version,omitempty" yaml:"version,omitempty"`
	Flags   *NvidiaGpuShareFlag   `json:"flags,omitempty" yaml:"flags,omitempty"`
	Sharing *NvidiaGpuShareDetail `json:"sharing,omitempty" yaml:"sharing,omitempty"`
}

type NvidiaGpuShareFlag struct {
	MIGStrategy string `json:"mig_strategy,omitempty" yaml:"migStrategy,omitempty"`
}

type NvidiaGpuShareDetail struct {
	TimeSlicing *NvidiaGpuShareTimeSlicing `json:"time_slicing,omitempty" yaml:"timeSlicing,omitempty"`
}

type NvidiaGpuShareTimeSlicing struct {
	Resources []NvidiaGpuShareTimeSlicingResource `json:"resources,omitempty" yaml:"resources,omitempty"`
}
type NvidiaGpuShareTimeSlicingResource struct {
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
	Replicas int    `json:"replicas,omitempty" yaml:"replicas,omitempty"`
}
