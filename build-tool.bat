@echo off
chcp 65001 > nul
echo.
echo ======================== BirthDayMemo 多平台编译工具 ========================
echo.
echo 请选择编译目标:
echo.
echo   1 - Windows x64
echo   2 - Windows ARM64
echo   3 - Linux x64
echo   4 - Linux ARM64
echo   5 - macOS x64 (Intel)
echo   6 - macOS ARM64 (Apple Silicon)
echo   7 - 全部平台
echo.
set /p choice=请输入选择 [1-7]: 
echo.

if "%choice%"=="1" cd frontend && npm run build && cd .. && set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build -o BDM\birthdaymemo-windows-amd64.exe .
if "%choice%"=="2" cd frontend && npm run build && cd .. && set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=arm64&& go build -o BDM\birthdaymemo-windows-arm64.exe .
if "%choice%"=="3" cd frontend && npm run build && cd .. && set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -o BDM\birthdaymemo-linux-amd64 .
if "%choice%"=="4" cd frontend && npm run build && cd .. && set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=arm64&& go build -o BDM\birthdaymemo-linux-arm64 .
if "%choice%"=="5" cd frontend && npm run build && cd .. && set CGO_ENABLED=0&& set GOOS=darwin&& set GOARCH=amd64&& go build -o BDM\birthdaymemo-macos-amd64 .
if "%choice%"=="6" cd frontend && npm run build && cd .. && set CGO_ENABLED=0&& set GOOS=darwin&& set GOARCH=arm64&& go build -o BDM\birthdaymemo-macos-arm64 .
if "%choice%"=="7" cd frontend && npm run build && cd .. && set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build -o BDM\birthdaymemo-windows-amd64.exe . && set GOOS=windows&& set GOARCH=arm64&& go build -o BDM\birthdaymemo-windows-arm64.exe . && set GOOS=linux&& set GOARCH=amd64&& go build -o BDM\birthdaymemo-linux-amd64 . && set GOOS=linux&& set GOARCH=arm64&& go build -o BDM\birthdaymemo-linux-arm64 . && set GOOS=darwin&& set GOARCH=amd64&& go build -o BDM\birthdaymemo-macos-amd64 . && set GOOS=darwin&& set GOARCH=arm64&& go build -o BDM\birthdaymemo-macos-arm64 .

echo.
echo 编译完成！
echo.
pause
