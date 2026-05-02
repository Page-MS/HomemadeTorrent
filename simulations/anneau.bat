@echo off

set N=%1

if "%N%"=="" (
  echo Usage: run_ring.bat ^<number_of_nodes^>
  exit /b 1
)

if %N% LSS 2 (
  echo Need at least 2 nodes
  exit /b 1
)

echo Starting ring with %N% nodes

setlocal EnableDelayedExpansion

set IDS=

for /L %%i in (1,1,%N%) do (
  set IDS=!IDS! Site%%i
)

for %%i in (!IDS!) do (
  echo Starting %%i
  start "%%i" cmd /k "title %%i && go run ..\src\homemadeTorrent\main.go %%i !IDS!"
)

echo All nodes started