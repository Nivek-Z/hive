@echo off
rem ============================================================
rem  One-click launcher: MySQL + Hive  ->  http://localhost:8080
rem ============================================================
setlocal
set BASE=%~dp0
if not defined JAVA_HOME set JAVA_HOME=D:\JDK-25
set JAR=%BASE%hive\target\hive.jar

call "%BASE%start-mysql.bat"
if errorlevel 1 exit /b 1

if not exist "%JAR%" (
    echo Building project - the first build may take a few minutes...
    call "%BASE%tools\apache-maven-3.9.16\bin\mvn.cmd" -q -DskipTests -f "%BASE%hive\pom.xml" package
    if errorlevel 1 (
        echo [ERROR] Build failed.
        pause
        exit /b 1
    )
)

echo Starting Hive at http://localhost:8080 ...
cd /d "%BASE%"
"%JAVA_HOME%\bin\java" -jar "%JAR%"
