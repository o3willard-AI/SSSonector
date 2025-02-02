; SSSonector Windows Installer Script
!include "MUI2.nsh"

; General
Name "SSSonector"
OutFile "..\build\sssonector-1.0.0-setup.exe"
InstallDir "$PROGRAMFILES64\SSSonector"
RequestExecutionLevel admin

; Interface Settings
!define MUI_ABORTWARNING

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; Languages
!insertmacro MUI_LANGUAGE "English"

Section "SSSonector" SecMain
    SetOutPath "$INSTDIR"
    
    ; Main executable
    File "..\build\win\sssonector.exe"
    
    ; Config files
    SetOutPath "$INSTDIR\configs"
    File /r "..\build\win\configs\*"
    
    ; Create config directory
    CreateDirectory "$PROGRAMDATA\SSSonector"
    CreateDirectory "$PROGRAMDATA\SSSonector\certs"
    CreateDirectory "$PROGRAMDATA\SSSonector\logs"
    
    ; Copy default configs
    CopyFiles "$INSTDIR\configs\*" "$PROGRAMDATA\SSSonector"
    
    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Create start menu shortcuts
    CreateDirectory "$SMPROGRAMS\SSSonector"
    CreateShortcut "$SMPROGRAMS\SSSonector\SSSonector.lnk" "$INSTDIR\sssonector.exe"
    CreateShortcut "$SMPROGRAMS\SSSonector\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    ; Add to PATH
    EnVar::AddValue "Path" "$INSTDIR"
    
    ; Register service
    ExecWait 'sc create "SSSonector" binPath= "$INSTDIR\sssonector.exe -config $PROGRAMDATA\SSSonector\config.yaml" start= auto'
    ExecWait 'sc description "SSSonector" "SSL tunneling service"'
SectionEnd

Section "Uninstall"
    ; Stop and remove service
    ExecWait 'sc stop "SSSonector"'
    ExecWait 'sc delete "SSSonector"'
    
    ; Remove from PATH
    EnVar::DeleteValue "Path" "$INSTDIR"
    
    ; Remove files
    Delete "$INSTDIR\sssonector.exe"
    RMDir /r "$INSTDIR\configs"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR"
    
    ; Remove start menu shortcuts
    Delete "$SMPROGRAMS\SSSonector\SSSonector.lnk"
    Delete "$SMPROGRAMS\SSSonector\Uninstall.lnk"
    RMDir "$SMPROGRAMS\SSSonector"
    
    ; Don't remove config/data directory by default
    MessageBox MB_YESNO "Do you want to remove all configuration files?" IDNO NoRemoveConfig
        RMDir /r "$PROGRAMDATA\SSSonector"
    NoRemoveConfig:
SectionEnd
