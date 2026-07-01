@echo off
rem ============================================================
rem  Manage portable MySQL 8.0 for Hive.
rem
rem  Usage:
rem      start-mysql.bat          Start MySQL
rem      start-mysql.bat status   Print MySQL status
rem      start-mysql.bat stop     Stop MySQL
rem
rem  Port configuration:
rem      1. HIVE_DB_PORT environment variable
rem      2. MYSQL_PORT in mysql.properties next to this script
rem      3. Default 3306
rem ============================================================
setlocal EnableExtensions EnableDelayedExpansion
set "BASE=%~dp0"
set "BASE=%BASE:~0,-1%"
set "CONFIG_FILE=%BASE%\mysql.properties"
set "PORT_FILE=%BASE%\data\mysql-port.txt"
set "ACTION=%~1"
set "MYSQL_PORT="

if "%ACTION%"=="" set "ACTION=start"
if /i not "%ACTION%"=="start" if /i not "%ACTION%"=="status" if /i not "%ACTION%"=="stop" (
    echo [ERROR] Unknown action: %ACTION%
    echo Usage: start-mysql.bat [start^|status^|stop]
    exit /b 2
)

call :resolveToolBase
if errorlevel 1 exit /b 1

call :setupLink
if errorlevel 1 exit /b 1

call :resolvePort
if errorlevel 1 exit /b 1

if /i "%ACTION%"=="status" goto status
if /i "%ACTION%"=="stop" goto stop
goto start

:start
if not exist "%BASE%\data" mkdir "%BASE%\data"

call :mysqlPing %MYSQL_PORT%
if not errorlevel 1 goto ready

call :portFree %MYSQL_PORT%
if errorlevel 1 (
    echo [ERROR] Port %MYSQL_PORT% is already in use, but portable MySQL is not responding.
    echo         Edit "%CONFIG_FILE%" or set HIVE_DB_PORT to a free port.
    exit /b 1
)

if not exist "%DATADIR%\mysql" (
    echo First run: initializing MySQL data directory, please wait...
    "%MYSQL_HOME%\bin\mysqld" --initialize-insecure --datadir="%DATADIR%" --console
    if errorlevel 1 exit /b 1
)

echo Starting MySQL on port %MYSQL_PORT% ...
start "hive-mysql-%MYSQL_PORT%" /min "%MYSQL_HOME%\bin\mysqld" --datadir="%DATADIR%" --port=%MYSQL_PORT% --console

set /a tries=0
:wait
set /a tries+=1
if %tries% gtr 30 (
    echo [ERROR] MySQL did not start in time on port %MYSQL_PORT%.
    exit /b 1
)
ping -n 2 127.0.0.1 >nul
call :mysqlPing %MYSQL_PORT%
if not errorlevel 1 goto ready
"%MYSQL_HOME%\bin\mysqladmin" --connect-timeout=1 -h127.0.0.1 -P%MYSQL_PORT% -uroot ping >nul 2>&1
if errorlevel 1 goto wait
rem Fresh initialization starts with an empty root password. Set the demo password once.
"%MYSQL_HOME%\bin\mysql" -h127.0.0.1 -P%MYSQL_PORT% -uroot --skip-password -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '123456'; FLUSH PRIVILEGES;" >nul 2>&1
goto wait

:ready
>"%PORT_FILE%" echo %MYSQL_PORT%
echo MySQL is ready on port %MYSQL_PORT% (user: root / password: 123456).
endlocal & set "HIVE_DB_HOST=127.0.0.1" & set "HIVE_DB_PORT=%MYSQL_PORT%" & exit /b 0

:status
call :mysqlPing %MYSQL_PORT%
if not errorlevel 1 (
    echo MySQL is running on port %MYSQL_PORT%.
    exit /b 0
)
call :portFree %MYSQL_PORT%
if errorlevel 1 (
    echo Port %MYSQL_PORT% is in use, but portable MySQL is not responding.
    exit /b 1
)
echo MySQL is not running on configured port %MYSQL_PORT%.
exit /b 1

:stop
call :mysqlPing %MYSQL_PORT%
if errorlevel 1 (
    echo MySQL is not running on configured port %MYSQL_PORT%.
    exit /b 0
)
echo Stopping MySQL on port %MYSQL_PORT% ...
"%MYSQL_HOME%\bin\mysqladmin" --connect-timeout=1 -h127.0.0.1 -P%MYSQL_PORT% -uroot -p123456 shutdown
if errorlevel 1 exit /b 1
echo MySQL stopped.
exit /b 0

:resolveToolBase
set "TOOL_BASE=%BASE%"
if exist "%TOOL_BASE%\tools\mysql-8.0.28-winx64\bin\mysqld.exe" exit /b 0
for %%I in ("%BASE%\..\..") do set "TOOL_BASE=%%~fI"
if exist "%TOOL_BASE%\tools\mysql-8.0.28-winx64\bin\mysqld.exe" exit /b 0
echo [ERROR] Portable MySQL tools not found under:
echo         %BASE%\tools
echo         %BASE%\..\..\tools
exit /b 1

:setupLink
set "LINK=%USERPROFILE%\.hive-link-member-packages-backend"
if not exist "%LINK%\tools\mysql-8.0.28-winx64\bin\mysqld.exe" (
    if exist "%LINK%" rmdir "%LINK%"
    mklink /J "%LINK%" "%TOOL_BASE%" >nul
    if errorlevel 1 (
        echo [ERROR] Could not create junction: %LINK%
        exit /b 1
    )
)
set "MYSQL_HOME=%LINK%\tools\mysql-8.0.28-winx64"
set "DATADIR=%LINK%\data\mysql"
if not exist "%MYSQL_HOME%\bin\mysqld.exe" (
    echo [ERROR] MySQL not found: %MYSQL_HOME%
    exit /b 1
)
exit /b 0

:resolvePort
if defined HIVE_DB_PORT (
    set "MYSQL_PORT=%HIVE_DB_PORT%"
) else if exist "%CONFIG_FILE%" (
    for /f "usebackq eol=# tokens=1,* delims==" %%A in ("%CONFIG_FILE%") do (
        if /i "%%~A"=="MYSQL_PORT" set "MYSQL_PORT=%%~B"
    )
)
if not defined MYSQL_PORT set "MYSQL_PORT=3306"
echo(%MYSQL_PORT%| findstr /r "^[0-9][0-9]*$" >nul
if errorlevel 1 (
    echo [ERROR] Invalid MySQL port: %MYSQL_PORT%
    exit /b 1
)
set /a PORT_NUM=%MYSQL_PORT%
if %PORT_NUM% lss 1 (
    echo [ERROR] Invalid MySQL port: %MYSQL_PORT%
    exit /b 1
)
if %PORT_NUM% gtr 65535 (
    echo [ERROR] Invalid MySQL port: %MYSQL_PORT%
    exit /b 1
)
exit /b 0

:mysqlPing
"%MYSQL_HOME%\bin\mysqladmin" --connect-timeout=1 -h127.0.0.1 -P%~1 -uroot -p123456 ping >nul 2>&1
exit /b %ERRORLEVEL%

:portFree
powershell -NoProfile -Command "$p=%~1; $l=$null; try { $l=[Net.Sockets.TcpListener]::new([Net.IPAddress]::Any,$p); $l.Start(); exit 0 } catch { exit 1 } finally { if ($l) { $l.Stop() } }" >nul 2>&1
exit /b %ERRORLEVEL%
