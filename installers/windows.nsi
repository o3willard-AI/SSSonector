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
    CreateDirectory "$APPDATA\SSSonector"
    CreateDirectory "$APPDATA\SSSonector\certs"
    CreateDirectory "$APPDATA\SSSonector\logs"
    
    ; Copy default configs
    CopyFiles "$INSTDIR\configs\*" "$APPDATA\SSSonector"
    
    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Create start menu shortcuts
    CreateDirectory "$SMPROGRAMS\SSSonector"
    CreateShortcut "$SMPROGRAMS\SSSonector\SSSonector.lnk" "$INSTDIR\sssonector.exe"
    CreateShortcut "$SMPROGRAMS\SSSonector\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    ; Register service
    ExecWait 'sc create "SSSonector" binPath= "$INSTDIR\sssonector.exe -config $APPDATA\SSSonector\config.yaml" start= auto'
    ExecWait 'sc description "SSSonector" "SSL tunneling service"'
    
    ; Write registry keys for uninstaller
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" "DisplayName" "SSSonector"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" "UninstallString" "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
    ; Stop and remove service
    ExecWait 'sc stop "SSSonector"'
    ExecWait 'sc delete "SSSonector"'
    
    ; Remove files
    Delete "$INSTDIR\sssonector.exe"
    RMDir /r "$INSTDIR\configs"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR"
    
    ; Remove start menu shortcuts
    Delete "$SMPROGRAMS\SSSonector\SSSonector.lnk"
    Delete "$SMPROGRAMS\SSSonector\Uninstall.lnk"
    RMDir "$SMPROGRAMS\SSSonector"
    
    ; Remove registry keys
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector"
    
    ; Don't remove config/data directory by default
    MessageBox MB_YESNO "Do you want to remove all configuration files?" IDNO NoRemoveConfig
        RMDir /r "$APPDATA\SSSonector"
    NoRemoveConfig:
SectionEnd
