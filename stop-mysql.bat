@echo off
rem Stop the portable MySQL instance
setlocal
set BASE=%~dp0
set MYSQL_HOME=%BASE%tools\mysql-8.0.28-winx64

"%MYSQL_HOME%\bin\mysqladmin" -h127.0.0.1 -P3306 -uroot -p123456 shutdown
if errorlevel 1 (
    echo Could not stop MySQL - is it running?
) else (
    echo MySQL stopped.
)
pause
