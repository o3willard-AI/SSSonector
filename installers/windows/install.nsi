; SSSonector Windows Installer Script
; NSIS (Nullsoft Scriptable Install System) script

!include "MUI2.nsh"
!include "FileFunc.nsh"

; General
Name "SSSonector"
OutFile "SSSonector-Setup.exe"
InstallDir "$PROGRAMFILES\SSSonector"
RequestExecutionLevel admin

; Interface Settings
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; Language
!insertmacro MUI_LANGUAGE "English"

Section "MainSection" SEC01
    SetOutPath "$INSTDIR"
    
    ; Main executable and configs
    File "..\build\windows-amd64\SSSonector.exe"
    File /r "..\configs\*"
    
    ; Create data directory
    CreateDirectory "$INSTDIR\certs"
    CreateDirectory "$INSTDIR\logs"
    
    ; Install TAP driver
    ExecWait '"$INSTDIR\tap-windows.exe" /S'
    
    ; Create service
    ExecWait 'sc create "SSSonector" binPath= "$INSTDIR\SSSonector.exe --config $INSTDIR\config.yaml" start= auto'
    ExecWait 'sc description "SSSonector" "Secure Scalable SSL Connector Service"'
    
    ; Start service
    ExecWait 'net start "SSSonector"'
    
    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Create Start Menu shortcuts
    CreateDirectory "$SMPROGRAMS\SSSonector"
    CreateShortCut "$SMPROGRAMS\SSSonector\SSSonector.lnk" "$INSTDIR\SSSonector.exe"
    CreateShortCut "$SMPROGRAMS\SSSonector\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    ; Registry entries
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" \
                     "DisplayName" "SSSonector"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector" \
                     "UninstallString" "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
    ; Stop and remove service
    ExecWait 'net stop "SSSonector"'
    ExecWait 'sc delete "SSSonector"'
    
    ; Remove TAP driver
    ExecWait '"$INSTDIR\tap-windows.exe" /S /U'
    
    ; Remove files
    Delete "$INSTDIR\SSSonector.exe"
    Delete "$INSTDIR\uninstall.exe"
    RMDir /r "$INSTDIR\configs"
    RMDir /r "$INSTDIR\certs"
    RMDir /r "$INSTDIR\logs"
    RMDir "$INSTDIR"
    
    ; Remove shortcuts
    Delete "$SMPROGRAMS\SSSonector\SSSonector.lnk"
    Delete "$SMPROGRAMS\SSSonector\Uninstall.lnk"
    RMDir "$SMPROGRAMS\SSSonector"
    
    ; Remove registry entries
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SSSonector"
SectionEnd
