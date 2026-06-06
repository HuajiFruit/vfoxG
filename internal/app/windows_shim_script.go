//go:build windows

package app

import "fmt"

func windowsShimScript(pluginName string, alias string, sdkPath string) string {
	alias = windowsSafeShimName(alias)
	return fmt.Sprintf(`@echo off
setlocal
set "SDK_ROOT=%[1]s"
set "ALIAS_NAME=%[2]s"
if exist "%%SDK_ROOT%%\%[2]s.exe" ("%%SDK_ROOT%%\%[2]s.exe" %%* & exit /b)
if exist "%%SDK_ROOT%%\bin\%[2]s.exe" ("%%SDK_ROOT%%\bin\%[2]s.exe" %%* & exit /b)
if exist "%%SDK_ROOT%%\Scripts\%[2]s.exe" ("%%SDK_ROOT%%\Scripts\%[2]s.exe" %%* & exit /b)
if exist "%%SDK_ROOT%%\sbin\%[2]s.exe" ("%%SDK_ROOT%%\sbin\%[2]s.exe" %%* & exit /b)
if exist "%%SDK_ROOT%%\%[2]s.cmd" (call "%%SDK_ROOT%%\%[2]s.cmd" %%* & exit /b)
if exist "%%SDK_ROOT%%\bin\%[2]s.cmd" (call "%%SDK_ROOT%%\bin\%[2]s.cmd" %%* & exit /b)
if exist "%%SDK_ROOT%%\Scripts\%[2]s.cmd" (call "%%SDK_ROOT%%\Scripts\%[2]s.cmd" %%* & exit /b)
if exist "%%SDK_ROOT%%\%[2]s.bat" (call "%%SDK_ROOT%%\%[2]s.bat" %%* & exit /b)
if exist "%%SDK_ROOT%%\bin\%[2]s.bat" (call "%%SDK_ROOT%%\bin\%[2]s.bat" %%* & exit /b)
if exist "%%SDK_ROOT%%\Scripts\%[2]s.bat" (call "%%SDK_ROOT%%\Scripts\%[2]s.bat" %%* & exit /b)
for /f "delims=" %%%%I in ('where "%%ALIAS_NAME%%" 2^>nul') do (
  if /I not "%%%%~fI"=="%%~f0" (
    if /I not "%%%%~dpI"=="%%~dp0" (
      if /I "%%%%~xI"==".cmd" (call "%%%%~fI" %%* & exit /b)
      if /I "%%%%~xI"==".bat" (call "%%%%~fI" %%* & exit /b)
      "%%%%~fI" %%*
      exit /b
    )
  )
)
echo vfoxG: %[2]s for %[3]s is not available under %%SDK_ROOT%%, and no fallback %[2]s was found on PATH. 1>&2
exit /b 9009
`, sdkPath, alias, pluginName)
}
