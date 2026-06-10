@echo off
rem ============================================================
rem  Start portable MySQL 8.0 (auto-initialize on first run)
rem
rem  NOTE: mysqld cannot parse non-ASCII (e.g. Chinese) paths on
rem  its command line, so we access the project through an ASCII
rem  NTFS junction at %USERPROFILE%\.hive-link instead.
rem ============================================================
setlocal
set BASE=%~dp0
set BASE=%BASE:~0,-1%
set LINK=%USERPROFILE%\.hive-link

rem Refresh the junction every run (survives folder moves/renames)
if exist "%LINK%" rmdir "%LINK%"
mklink /J "%LINK%" "%BASE%" >nul
if errorlevel 1 (
    echo [ERROR] Could not create junction: %LINK%
    pause
    exit /b 1
)

set MYSQL_HOME=%LINK%\tools\mysql-8.0.28-winx64
set DATADIR=%LINK%\data\mysql

if not exist "%MYSQL_HOME%\bin\mysqld.exe" (
    echo [ERROR] MySQL not found: %MYSQL_HOME%
    pause
    exit /b 1
)

rem Already running?
"%MYSQL_HOME%\bin\mysqladmin" -h127.0.0.1 -P3306 -uroot -p123456 ping >nul 2>&1
if not errorlevel 1 (
    echo MySQL is already running.
    exit /b 0
)

if not exist "%DATADIR%\mysql" (
    echo First run: initializing MySQL data directory, please wait...
    "%MYSQL_HOME%\bin\mysqld" --initialize-insecure --datadir="%DATADIR%" --console
)

echo Starting MySQL on port 3306 ...
start "hive-mysql" /min "%MYSQL_HOME%\bin\mysqld" --datadir="%DATADIR%" --port=3306 --console

set /a tries=0
:wait
set /a tries+=1
if %tries% gtr 30 (
    echo [ERROR] MySQL did not start in time.
    pause
    exit /b 1
)
ping -n 2 127.0.0.1 >nul
"%MYSQL_HOME%\bin\mysqladmin" -h127.0.0.1 -P3306 -uroot -p123456 ping >nul 2>&1
if not errorlevel 1 goto ready
"%MYSQL_HOME%\bin\mysqladmin" -h127.0.0.1 -P3306 -uroot ping >nul 2>&1
if errorlevel 1 goto wait
rem Server is up with empty root password (fresh init) - set it to 123456
"%MYSQL_HOME%\bin\mysql" -h127.0.0.1 -uroot --skip-password -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '123456'; FLUSH PRIVILEGES;" >nul 2>&1

:ready
echo MySQL is ready (user: root / password: 123456).
exit /b 0
