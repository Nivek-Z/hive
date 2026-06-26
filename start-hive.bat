@echo off
rem ============================================================
rem  One-click launcher: MySQL + Hive with dynamic ports.
rem
rem  Defaults: MySQL 3306, Hive 8080. If occupied, the scripts
rem  try the next free ports and print the final URL.
rem ============================================================
setlocal EnableExtensions
set "BASE=%~dp0"
set "BASE=%BASE:~0,-1%"
if not defined JAVA_HOME set "JAVA_HOME=D:\JDK-25"
set "JAR=%BASE%\hive\target\hive.jar"
set "SERVER_PORT_FILE=%BASE%\data\server-port.txt"

call :resolveToolBase
if errorlevel 1 exit /b 1

call "%BASE%\start-mysql.bat"
if errorlevel 1 exit /b 1

if not defined HIVE_DB_PORT (
    echo [ERROR] start-mysql.bat did not provide HIVE_DB_PORT.
    pause
    exit /b 1
)

if not exist "%JAR%" (
    echo Building project - the first build may take a few minutes...
    call "%MAVEN_CMD%" -q -DskipTests -f "%BASE%\hive\pom.xml" package
    if errorlevel 1 (
        echo [ERROR] Build failed.
        pause
        exit /b 1
    )
)

if not defined SERVER_PORT call :findFreePort 8080 8999 SERVER_PORT
if not defined SERVER_PORT (
    echo [ERROR] No free Hive server port found in range 8080-8999.
    pause
    exit /b 1
)

if not exist "%BASE%\data" mkdir "%BASE%\data"
>"%SERVER_PORT_FILE%" echo %SERVER_PORT%

echo MySQL port: %HIVE_DB_PORT%
echo Starting Hive at http://localhost:%SERVER_PORT% ...
cd /d "%BASE%"
"%JAVA_HOME%\bin\java" -jar "%JAR%"
exit /b %ERRORLEVEL%

:resolveToolBase
set "TOOL_BASE=%BASE%"
if exist "%TOOL_BASE%\tools\apache-maven-3.9.16\bin\mvn.cmd" goto foundTools
for %%I in ("%BASE%\..\..") do set "TOOL_BASE=%%~fI"
if exist "%TOOL_BASE%\tools\apache-maven-3.9.16\bin\mvn.cmd" goto foundTools
echo [ERROR] Portable Maven tools not found under:
echo         %BASE%\tools
echo         %BASE%\..\..\tools
pause
exit /b 1
:foundTools
set "MAVEN_CMD=%TOOL_BASE%\tools\apache-maven-3.9.16\bin\mvn.cmd"
exit /b 0

:findFreePort
set "%~3="
for /f %%P in ('powershell -NoProfile -ExecutionPolicy Bypass -File "%BASE%\scripts\find-free-port.ps1" %~1 %~2') do set "%~3=%%P"
exit /b 0
