@echo off
rem Rebuild the project (runs unit tests too)
setlocal
set BASE=%~dp0
if not defined JAVA_HOME set JAVA_HOME=D:\JDK-25
call "%BASE%tools\apache-maven-3.9.16\bin\mvn.cmd" -f "%BASE%hive\pom.xml" package %*
pause
