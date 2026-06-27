@echo off
setlocal

set BINARY=sarde
set DIST=dist
set MODULE=github.com/getsarde/sarde

for /f "delims=" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i
if "%VERSION%"=="" set VERSION=dev

if "%1"==""        goto build
if "%1"=="build"   goto build
if "%1"=="release" goto release
if "%1"=="test"    goto test
if "%1"=="bench"   goto bench
if "%1"=="vet"     goto vet
if "%1"=="clean"   goto clean
echo Unknown target: %1
exit /b 1

:build
if not exist %DIST% mkdir %DIST%
go build -ldflags "-X %MODULE%/internal/version.Version=%VERSION%" -o %DIST%\%BINARY%.exe .\cmd\sarde
goto end

:release
if not exist %DIST% mkdir %DIST%
set GOOS=linux&  set GOARCH=amd64& go build -ldflags "-X %MODULE%/internal/version.Version=%VERSION%" -o %DIST%\%BINARY%-linux-amd64    .\cmd\sarde
set GOOS=darwin& set GOARCH=amd64& go build -ldflags "-X %MODULE%/internal/version.Version=%VERSION%" -o %DIST%\%BINARY%-darwin-amd64   .\cmd\sarde
set GOOS=darwin& set GOARCH=arm64& go build -ldflags "-X %MODULE%/internal/version.Version=%VERSION%" -o %DIST%\%BINARY%-darwin-arm64   .\cmd\sarde
set GOOS=windows&set GOARCH=amd64& go build -ldflags "-X %MODULE%/internal/version.Version=%VERSION%" -o %DIST%\%BINARY%-windows-amd64.exe .\cmd\sarde
goto end

:test
go test .\...
goto end

:bench
go test -bench=. -benchmem -timeout 300s .\internal\build\
goto end

:vet
go vet .\...
goto end

:clean
if exist %DIST%  rmdir /s /q %DIST%
if exist .cache  rmdir /s /q .cache
goto end

:end
endlocal
