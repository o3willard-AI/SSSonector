//go:build windows

package provision

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// restrictKeyFileWindows replaces the DACL with one granting full access to
// SYSTEM, Administrators, and the file owner only.
func restrictKeyFileWindows(path string) error {
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;BA)(A;OICI;FA;;;SY)(A;OICI;FA;;;OW)")
	if err != nil {
		return fmt.Errorf("provision: acl descriptor: %w", err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("provision: acl extract: %w", err)
	}
	const daclSecurityInfo = 4 // DACL_SECURITY_INFORMATION
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, daclSecurityInfo, nil, nil, dacl, nil); err != nil {
		return fmt.Errorf("provision: SetNamedSecurityInfo: %w (default ProgramData ACL still applies)", err)
	}
	return nil
}
