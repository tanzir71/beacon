#ifndef SourceDir
  #define SourceDir "..\..\bin"
#endif
#ifndef OutputDir
  #define OutputDir "..\..\dist\installer"
#endif
#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif
#ifndef NumericVersion
  #define NumericVersion "0.0.0.0"
#endif

[Setup]
AppId={{49BAE7EA-599A-4F46-A24B-7905900A85E9}
AppName=Beacon
AppVersion={#AppVersion}
AppPublisher=Beacon contributors
AppPublisherURL=https://github.com/tanzir71/beacon
AppSupportURL=https://github.com/tanzir71/beacon/issues
AppUpdatesURL=https://github.com/tanzir71/beacon/releases/latest
DefaultDirName={autopf}\Beacon
DefaultGroupName=Beacon
DisableProgramGroupPage=yes
PrivilegesRequired=admin
PrivilegesRequiredOverridesAllowed=commandline
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir={#OutputDir}
OutputBaseFilename=beacon-{#AppVersion}-windows-amd64-setup
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
UninstallDisplayIcon={app}\beacon.exe
LicenseFile=..\..\LICENSE
VersionInfoVersion={#NumericVersion}
VersionInfoCompany=Beacon contributors
VersionInfoDescription=Beacon local app server installer
VersionInfoProductName=Beacon
VersionInfoProductVersion={#NumericVersion}

[Files]
Source: "{#SourceDir}\beacon.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\beacon-cli.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Beacon"; Filename: "{app}\beacon.exe"
Name: "{group}\Beacon diagnostics"; Filename: "{app}\beacon-cli.exe"; Parameters: "doctor"

[Run]
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""Beacon"""; Flags: runhidden waituntilterminated; Check: IsAdminInstallMode
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall add rule name=""Beacon"" dir=in action=allow program=""{app}\beacon.exe"" enable=yes profile=private"; Flags: runhidden waituntilterminated; Check: IsAdminInstallMode
Filename: "{app}\beacon.exe"; Description: "Start Beacon"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{sys}\taskkill.exe"; Parameters: "/IM beacon.exe /T /F"; Flags: runhidden waituntilterminated; RunOnceId: "StopBeacon"
Filename: "{sys}\taskkill.exe"; Parameters: "/IM beacon-cli.exe /T /F"; Flags: runhidden waituntilterminated; RunOnceId: "StopBeaconCLI"
Filename: "{app}\beacon-cli.exe"; Parameters: "autostart disable"; Flags: runhidden waituntilterminated; RunOnceId: "RemoveAutostart"
Filename: "{app}\beacon-cli.exe"; Parameters: "trust --uninstall"; Flags: runhidden waituntilterminated; RunOnceId: "RemoveTrust"
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""Beacon"""; Flags: runhidden waituntilterminated; RunOnceId: "RemoveFirewall"

[UninstallDelete]
; Beacon owns these two per-user directories. The user's
; {userprofile}\Beacon\Apps directory is intentionally outside this list.
Type: filesandordirs; Name: "{localappdata}\Beacon"
Type: filesandordirs; Name: "{userappdata}\Beacon"
