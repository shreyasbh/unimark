package target

import "time"

// BootPhases captures the time breakdown of each boot phase
type BootPhases struct {
	// Docker phases
	ImageLoad       time.Duration
	ContainerCreate time.Duration
	NetworkConnect  time.Duration
	ProcessStart    time.Duration

	// Nanos phases
	QEMUStartup time.Duration
	KernelBoot  time.Duration
	NetworkUp   time.Duration
	AppReady    time.Duration

	// Total end to end
	Total time.Duration
}
