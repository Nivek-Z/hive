@echo off
rem ============================================================
rem  Start portable MySQL 8.0 with dynamic port selection.
rem
rem  Default port: 3306. If occupied, the script tries 3307-3999.
rem  It exports HIVE_DB_HOST/HIVE_DB_PORT back to the caller.
rem ============================================================
setlocal EnableExtensions EnableDelayedExpansion
set "BASE=%~dp0"
set "BASE=%BASE:~0,-1%"
set "MYSQL_PORT="
set "PORT_FILE=%BASE%\data\mysql-port.txt"

call :resolveToolBase
if errorlevel 1 exit /b 1

set "LINK=%USERPROFILE%\.hive-link-member-packages-backend"
if not exist "%LINK%\tools\mysql-8.0.28-winx64\bin\mysqld.exe" (
    if exist "%LINK%" rmdir "%LINK%"
    mklink /J "%LINK%" "%TOOL_BASE%" >nul
    if errorlevel 1 (
        echo [ERROR] Could not create junction: %LINK%
        pause
        exit /b 1
    )
)

set "MYSQL_HOME=%LINK%\tools\mysql-8.0.28-winx64"
set "DATADIR=%LINK%\data\mysql"

if not exist "%MYSQL_HOME%\bin\mysqld.exe" (
    echo [ERROR] MySQL not found: %MYSQL_HOME%
    pause
    exit /b 1
)

if not exist "%BASE%\data" mkdir "%BASE%\data"

if defined HIVE_DB_PORT (
    set "MYSQL_PORT=%HIVE_DB_PORT%"
    call :mysqlPing %MYSQL_PORT%
    if not errorlevel 1 goto ready
    call :findFreePort %MYSQL_PORT% %MYSQL_PORT% REQUESTED_FREE_PORT
    if not "!REQUESTED_FREE_PORT!"=="!MYSQL_PORT!" (
        echo [ERROR] Requested MySQL port %MYSQL_PORT% is already in use.
        pause
        exit /b 1
    )
) else (
    call :findRunningMysql
    if not defined MYSQL_PORT call :findFreePort 3306 3999 MYSQL_PORT
)

if not defined MYSQL_PORT (
    echo [ERROR] No free MySQL port found in range 3306-3999.
    pause
    exit /b 1
)

call :mysqlPing %MYSQL_PORT%
if not errorlevel 1 goto ready

if not exist "%DATADIR%\mysql" (
    echo First run: initializing MySQL data directory, please wait...
    "%MYSQL_HOME%\bin\mysqld" --initialize-insecure --datadir="%DATADIR%" --console
)

echo Starting MySQL on port %MYSQL_PORT% ...
start "hive-mysql-%MYSQL_PORT%" /min "%MYSQL_HOME%\bin\mysqld" --datadir="%DATADIR%" --port=%MYSQL_PORT% --console

set /a tries=0
:wait
set /a tries+=1
if %tries% gtr 30 (
    echo [ERROR] MySQL did not start in time on port %MYSQL_PORT%.
    pause
    exit /b 1
)
ping -n 2 127.0.0.1 >nul
call :mysqlPing %MYSQL_PORT%
if not errorlevel 1 goto ready
"%MYSQL_HOME%\bin\mysqladmin" --connect-timeout=1 -h127.0.0.1 -P%MYSQL_PORT% -uroot ping >nul 2>&1
if errorlevel 1 goto wait
rem Server is up with empty root password (fresh init) - set it to 123456.
"%MYSQL_HOME%\bin\mysql" -h127.0.0.1 -P%MYSQL_PORT% -uroot --skip-password -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '123456'; FLUSH PRIVILEGES;" >nul 2>&1

goto wait

:ready
>"%PORT_FILE%" echo %MYSQL_PORT%
echo MySQL is ready on port %MYSQL_PORT% (user: root / password: 123456).
endlocal & set "HIVE_DB_HOST=127.0.0.1" & set "HIVE_DB_PORT=%MYSQL_PORT%" & exit /b 0

:resolveToolBase
set "TOOL_BASE=%BASE%"
if exist "%TOOL_BASE%\tools\mysql-8.0.28-winx64\bin\mysqld.exe" exit /b 0
for %%I in ("%BASE%\..\..") do set "TOOL_BASE=%%~fI"
if exist "%TOOL_BASE%\tools\mysql-8.0.28-winx64\bin\mysqld.exe" exit /b 0
echo [ERROR] Portable MySQL tools not found under:
echo         %BASE%\tools
echo         %BASE%\..\..\tools
pause
exit /b 1

:mysqlPing
"%MYSQL_HOME%\bin\mysqladmin" --connect-timeout=1 -h127.0.0.1 -P%~1 -uroot -p123456 ping >nul 2>&1
exit /b %ERRORLEVEL%

:findRunningMysql
for /l %%P in (3306,1,3316) do (
    call :mysqlPing %%P
    if not errorlevel 1 (
        set "MYSQL_PORT=%%P"
        exit /b 0
    )
)
exit /b 1

:findFreePort
set "%~3="
for /f %%P in ('powershell -NoProfile -ExecutionPolicy Bypass -File "%BASE%\scripts\find-free-port.ps1" %~1 %~2') do set "%~3=%%P"
exit /b 0
