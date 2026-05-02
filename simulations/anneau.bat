@echo off

set N=3

setlocal EnableDelayedExpansion

set IDS=

for /L %%i in (1,1,%N%) do (
    set IDS=!IDS! %%i
)

echo Nodes: !IDS!

set CMDLINE=

for /L %%i in (1,1,%N%) do (
    if %%i==1 (
        set CMDLINE=go run ..\src\homemadeTorrent\main.go %%i!IDS!
    ) else (
        set CMDLINE=!CMDLINE! ^| go run ..\src\homemadeTorrent\main.go %%i!IDS!
    )
)

echo Running:
echo !CMDLINE!

cmd /k "!CMDLINE!"