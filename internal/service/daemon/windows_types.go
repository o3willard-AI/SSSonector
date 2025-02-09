//go:build windows
// +build windows

package daemon

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// JobObjectBasicLimitInformation represents basic Job Object limits
type JobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit uint64
	PerJobUserTimeLimit     uint64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

// JobObjectExtendedLimitInformation represents extended Job Object limits
type JobObjectExtendedLimitInformation struct {
	BasicLimitInformation JobObjectBasicLimitInformation
	IoInfo                IoCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// IoCounters represents I/O statistics
type IoCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

// ServiceStatus represents Windows service status
type ServiceStatus struct {
	ServiceType             uint32
	CurrentState            uint32
	ControlsAccepted        uint32
	Win32ExitCode           uint32
	ServiceSpecificExitCode uint32
	CheckPoint              uint32
	WaitHint                uint32
}

// ServiceConfig represents Windows service configuration
type ServiceConfig struct {
	ServiceType      uint32
	StartType        uint32
	ErrorControl     uint32
	BinaryPathName   *uint16
	LoadOrderGroup   *uint16
	TagId            uint32
	Dependencies     *uint16
	ServiceStartName *uint16
	DisplayName      *uint16
	Password         *uint16
	Description      *uint16
}

// JobObject represents a Windows Job Object
type JobObject struct {
	handle windows.Handle
}

// NewJobObject creates a new Job Object
func NewJobObject(name string) (*JobObject, error) {
	jobName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	handle, err := windows.CreateJobObject(nil, jobName)
	if err != nil {
		return nil, err
	}

	return &JobObject{handle: handle}, nil
}

// Close closes the Job Object handle
func (j *JobObject) Close() error {
	return windows.CloseHandle(j.handle)
}

// SetBasicLimits sets basic Job Object limits
func (j *JobObject) SetBasicLimits(limits *JobObjectBasicLimitInformation) error {
	return windows.SetInformationJobObject(
		j.handle,
		windows.JobObjectBasicLimitInformation,
		uintptr(unsafe.Pointer(limits)),
		uint32(unsafe.Sizeof(*limits)),
	)
}

// SetExtendedLimits sets extended Job Object limits
func (j *JobObject) SetExtendedLimits(limits *JobObjectExtendedLimitInformation) error {
	return windows.SetInformationJobObject(
		j.handle,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(limits)),
		uint32(unsafe.Sizeof(*limits)),
	)
}

// AssignProcess assigns a process to the Job Object
func (j *JobObject) AssignProcess(pid uint32) error {
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	return windows.AssignProcessToJobObject(j.handle, handle)
}

// TerminateJobObject terminates all processes in the Job Object
func (j *JobObject) TerminateJobObject(exitCode uint32) error {
	return windows.TerminateJobObject(j.handle, exitCode)
}

// GetBasicAccountingInformation gets basic Job Object accounting information
func (j *JobObject) GetBasicAccountingInformation() (*JobObjectBasicLimitInformation, error) {
	var info JobObjectBasicLimitInformation
	err := windows.QueryInformationJobObject(
		j.handle,
		windows.JobObjectBasicLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// GetExtendedAccountingInformation gets extended Job Object accounting information
func (j *JobObject) GetExtendedAccountingInformation() (*JobObjectExtendedLimitInformation, error) {
	var info JobObjectExtendedLimitInformation
	err := windows.QueryInformationJobObject(
		j.handle,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
