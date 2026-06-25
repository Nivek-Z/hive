@echo off
rem Stop the portable MySQL instance started by this worktree.
setlocal EnableExtensions
set "BASE=%~dp0"
set "BASE=%BASE:~0,-1%"
set "PORT_FILE=%BASE%\data\mysql-port.txt"

call :resolveToolBase
if errorlevel 1 exit /b 1

if not defined HIVE_DB_PORT if exist "%PORT_FILE%" set /p HIVE_DB_PORT=<"%PORT_FILE%"

if defined HIVE_DB_PORT (
    call :shutdownPort %HIVE_DB_PORT%
    goto done
)

for /l %%P in (3306,1,3999) do (
    "%MYSQL_HOME%\bin\mysqladmin" --connect-timeout=1 -h127.0.0.1 -P%%P -uroot -p123456 ping >nul 2>&1
    if not errorlevel 1 (
        call :shutdownPort %%P
        goto done
    )
)

echo Could not find a running portable MySQL instance in range 3306-3999.
goto finish

:done
if errorlevel 1 (
    echo Could not stop MySQL on port %HIVE_DB_PORT% - is it running?
) else (
    echo MySQL stopped.
)

:finish
pause
exit /b 0

:resolveToolBase
set "TOOL_BASE=%BASE%"
if exist "%TOOL_BASE%\tools\mysql-8.0.28-winx64\bin\mysqladmin.exe" goto foundTools
for %%I in ("%BASE%\..\..") do set "TOOL_BASE=%%~fI"
if exist "%TOOL_BASE%\tools\mysql-8.0.28-winx64\bin\mysqladmin.exe" goto foundTools
echo [ERROR] Portable MySQL tools not found under:
echo         %BASE%\tools
echo         %BASE%\..\..\tools
pause
exit /b 1
:foundTools
set "MYSQL_HOME=%TOOL_BASE%\tools\mysql-8.0.28-winx64"
exit /b 0

:shutdownPort
set "HIVE_DB_PORT=%~1"
"%MYSQL_HOME%\bin\mysqladmin" --connect-timeout=1 -h127.0.0.1 -P%~1 -uroot -p123456 shutdown
exit /b %ERRORLEVEL%
