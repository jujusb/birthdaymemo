@echo off
setlocal EnableDelayedExpansion

set ROOT=%~dp0
pushd "%ROOT%" 2>nul

echo.
echo ============================================================
echo  BirthDayMemo - Clean Intermediate Files
echo ============================================================
echo.

if exist "frontend\dist" (
    echo [1] Removing frontend\dist\
    rmdir /S /Q "frontend\dist" 2>nul
) else (
    echo [1] frontend\dist\ not found
)

if exist "frontend\node_modules\.vite" (
    echo [2] Removing frontend\node_modules\.vite\
    rmdir /S /Q "frontend\node_modules\.vite" 2>nul
) else (
    echo [2] frontend\node_modules\.vite\ not found
)

if exist "frontend\node_modules\.cache" (
    echo [3] Removing frontend\node_modules\.cache\
    rmdir /S /Q "frontend\node_modules\.cache" 2>nul
) else (
    echo [3] frontend\node_modules\.cache\ not found
)

if exist "frontend\.tsbuildinfo" (
    echo [4] Removing frontend\.tsbuildinfo
    del /F /Q "frontend\.tsbuildinfo" 2>nul
)

if exist "coverage.out" (
    echo [5] Removing coverage.out
    del /F /Q "coverage.out" 2>nul
)

if exist "coverage.html" (
    echo [6] Removing coverage.html
    del /F /Q "coverage.html" 2>nul
)

for %%E in ( *.exe.tmp *.test *.out ) do (
    if exist "%%E" (
        echo [7] Removing %%E
        del /F /Q "%%E" 2>nul
    )
)

if exist "birthdaymemo.exe" (
    echo [8] Removing birthdaymemo.exe
    del /F /Q "birthdaymemo.exe" 2>nul
)

for %%P in ( *.tmp *.swp *.bak ) do (
    if exist "%%P" (
        echo [9] Removing %%P
        del /F /Q "%%P" 2>nul
    )
)

if exist "Thumbs.db" (
    echo [10] Removing Thumbs.db
    del /F /Q "Thumbs.db" 2>nul
)

if exist ".DS_Store" (
    echo [11] Removing .DS_Store
    del /F /Q ".DS_Store" 2>nul
)

echo.
echo ============================================================
echo  Clean completed
echo ============================================================
echo.
echo  Preserved:
echo    - Source code: internal/, main.go, frontend/src/
echo    - Dependencies: frontend/node_modules/
echo    - Final binary: BDM/birthdaymemo.exe
echo    - Runtime data: BDM/birthdaymemo.db, BDM/config.json, BDM/logs/
echo.

popd
endlocal
pause