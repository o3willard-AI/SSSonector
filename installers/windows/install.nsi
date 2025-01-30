; SSSonector Windows Installer Script

!include "MUI2.nsh"
!include "LogicLib.nsh"

; Installer attributes
Name "SSSonector"
OutFile "${BUILD_DIR}/installers/sssonector-${VERSION}-setup.exe"
InstallDir "$PROGRAMFILES64\SSSonector"
RequestExecutionLevel admin

; Interface settings
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "${SOURCE_DIR}/LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Languages
!insertmacro MUI_LANGUAGE "English"

Section "SSSonector" SecSSSonector
    SetOutPath "$INSTDIR"
    
    ; Install files
    File "${SOURCE_DIR}/sssonector.exe"
    File "${SOURCE_DIR}/*.yaml"
    File "${SOURCE_DIR}/install-service.ps1"
    
    ; Create directories
    CreateDirectory "$INSTDIR\certs"
    CreateDirectory "$APPDATA\SSSonector\logs"
    
    ; Create config directory and copy config files
    CreateDirectory "$APPDATA\SSSonector"
    CopyFiles "$INSTDIR\*.yaml" "$APPDATA\SSSonector"
    
    ; Set permissions
    nsExec::ExecToLog 'icacls "$APPDATA\SSSonector" /inheritance:r /grant:r "NT AUTHORITY\SYSTEM":(OI)(CI)F /grant:r "BUILTIN\Administrators":(OI)(CI)F'
    nsExec::ExecToLog 'icacls "$APPDATA\SSSonector\logs" /inheritance:r /grant:r "NT AUTHORITY\SYSTEM":(OI)(CI)F /grant:r "BUILTIN\Administrators":(OI)(CI)F'
    
    ; Install service
    nsExec::ExecToLog 'powershell -ExecutionPolicy Bypass -File "$INSTDIR\install-service.ps1"'
    
    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Create start menu shortcuts
    CreateDirectory "$SMPROGRAMS\SSSonector"
    CreateShortCut "$SMPROGRAMS\SSSonector\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    ; Add uninstall information to Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" \
                     "DisplayName" "SSSonector"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" \
                     "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" \
                     "DisplayIcon" "$INSTDIR\sssonector.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" \
                     "DisplayVersion" "${VERSION}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" \
                     "Publisher" "O3Willard"
SectionEnd

Section "Uninstall"
    ; Stop and remove service
    nsExec::ExecToLog 'net stop SSSonector'
    nsExec::ExecToLog 'sc delete SSSonector'
    
    ; Remove files
    Delete "$INSTDIR\sssonector.exe"
    Delete "$INSTDIR\*.yaml"
    Delete "$INSTDIR\install-service.ps1"
    Delete "$INSTDIR\uninstall.exe"
    
    ; Remove directories
    RMDir /r "$INSTDIR\certs"
    RMDir /r "$APPDATA\SSSonector"
    RMDir "$INSTDIR"
    
    ; Remove start menu shortcuts
    Delete "$SMPROGRAMS\SSSonector\Uninstall.lnk"
    RMDir "$SMPROGRAMS\SSSonector"
    
    ; Remove uninstall information
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector"
SectionEnd
