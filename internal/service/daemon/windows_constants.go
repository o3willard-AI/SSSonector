//go:build windows
// +build windows

package daemon

// Windows-specific constants for service management
const (
	// Service configuration
	ServiceName        = "SSonector"
	ServiceDisplayName = "SSonector Service"
	ServiceDescription = "SSonector secure communication service"

	// Service dependencies (null-terminated strings)
	ServiceDependencies = "Tcpip;Dnscache"

	// Service start type
	ServiceAutoStart = 2

	// Service error control
	ServiceErrorNormal = 1

	// Service accepted controls
	ServiceAcceptStop                  = 1
	ServiceAcceptPauseAndContinue      = 2
	ServiceAcceptShutdown              = 4
	ServiceAcceptParamChange           = 8
	ServiceAcceptNetBindChange         = 16
	ServiceAcceptHardwareProfileChange = 32
	ServiceAcceptPowerEvent            = 64
	ServiceAcceptSessionChange         = 128

	// Service control codes
	ServiceControlStop           = 1
	ServiceControlPause          = 2
	ServiceControlContinue       = 3
	ServiceControlInterrogate    = 4
	ServiceControlShutdown       = 5
	ServiceControlParamChange    = 6
	ServiceControlNetBindAdd     = 7
	ServiceControlNetBindRemove  = 8
	ServiceControlNetBindEnable  = 9
	ServiceControlNetBindDisable = 10

	// Service states
	ServiceStopped         = 1
	ServiceStartPending    = 2
	ServiceStopPending     = 3
	ServiceRunning         = 4
	ServiceContinuePending = 5
	ServicePausePending    = 6
	ServicePaused          = 7

	// Job Object limits
	JobObjectLimitProcessMemory = 0x00000100
	JobObjectLimitProcessTime   = 0x00000002
	JobObjectLimitActiveProcess = 0x00000008
	JobObjectLimitPriorityClass = 0x00000020
	JobObjectLimitAffinity      = 0x00000010
	JobObjectLimitWorkingSet    = 0x00000001

	// Job Object notification limits
	JobObjectLimitDieOnUnhandledException = 0x00000400
	JobObjectLimitBreakawayOK             = 0x00000800
	JobObjectLimitSilentBreakawayOK       = 0x00001000
	JobObjectLimitKillOnJobClose          = 0x00002000

	// Job Object security limits
	JobObjectSecurityNoAdmin         = 0x00000001
	JobObjectSecurityRestrictedToken = 0x00000002
	JobObjectSecurityOnlyToken       = 0x00000004
	JobObjectSecurityFilterTokens    = 0x00000008

	// Job Object UI limits
	JobObjectUILimitHandles          = 0x00000001
	JobObjectUILimitReadClipboard    = 0x00000002
	JobObjectUILimitWriteClipboard   = 0x00000004
	JobObjectUILimitSystemParameters = 0x00000008
	JobObjectUILimitDisplaySettings  = 0x00000010
	JobObjectUILimitGlobalAtoms      = 0x00000020
	JobObjectUILimitDesktop          = 0x00000040
	JobObjectUILimitExitWindows      = 0x00000080

	// Network isolation
	NetworkIsolationNone    = 0
	NetworkIsolationPrivate = 1
	NetworkIsolationPublic  = 2
	NetworkIsolationCustom  = 3
)

// Windows error codes
const (
	ErrorSuccess               = 0
	ErrorInvalidFunction       = 1
	ErrorFileNotFound          = 2
	ErrorPathNotFound          = 3
	ErrorAccessDenied          = 5
	ErrorInvalidHandle         = 6
	ErrorNotEnoughMemory       = 8
	ErrorInvalidData           = 13
	ErrorInvalidParameter      = 87
	ErrorBrokenPipe            = 109
	ErrorInsufficientBuffer    = 122
	ErrorServiceExists         = 1073
	ErrorServiceAlreadyRunning = 1056
	ErrorServiceNotActive      = 1062
	ErrorServiceRequestTimeout = 1053
)
