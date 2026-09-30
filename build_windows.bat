@echo off
REM ====================================================================
REM  OE WATCHER // Windows Native Build Script
REM ====================================================================

echo [1/2] OE Watcher Windows x64 derleniyor...
if not exist "dist" mkdir dist
go build -ldflags="-s -w" -o dist\oe-watcher.exe .

if %ERRORLEVEL% EQU 0 (
    echo [2/2] Yapilandirma ve veri dosyalari kopyalaniyor...
    if exist "config.json" copy /Y config.json dist\ >nul
    if exist "data" xcopy /E /I /Y data dist\data >nul
    echo.
    echo ====================================================================
    echo BASARILI: OE Watcher Windows paketi hazir!
    echo Calistirmak icin: dist\oe-watcher.exe
    echo ====================================================================
) else (
    echo HATA: Derleme basarisiz oldu. Lutfen Go ortamini kontrol edin.
)
