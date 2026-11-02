@echo off
REM ====================================================================
REM  OE WATCHER // Windows Kolay Kurulum ve Baslatma Scripti
REM ====================================================================
chcp 65001 >nul
cls

echo ====================================================================
echo   OE WATCHER // Windows Kurulum Asistani
echo ====================================================================
echo.

echo [1/4] OE Watcher derleniyor ve hazirlaniyor...
call build_windows.bat
if %ERRORLEVEL% NEQ 0 (
    echo [HATA] Derleme basarisiz oldu. Lutfen Go ortamini kontrol edin.
    pause
    exit /b 1
)

echo.
echo [2/4] Windows Baslangicina (Startup Run) ekleniyor...
set "TARGET_EXE=%~dp0dist\oe-watcher.exe"
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "OEWatcher" /t REG_SZ /d "\"%TARGET_EXE%\"" /f >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo   [OK] Windows her acildiginda otomatik baslatilacak.
) else (
    echo   [UYARI] Baslangic kaydi eklenemedi, manuel calistirabilirsiniz.
)

echo.
echo [3/4] Masaustune kisayol olusturuluyor...
set "SHORTCUT_PATH=%USERPROFILE%\Desktop\OE Watcher.lnk"
powershell -NoProfile -Command "$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%SHORTCUT_PATH%'); $s.TargetPath = '%TARGET_EXE%'; $s.WorkingDirectory = '%~dp0dist'; $s.Description = 'OE Watcher // Medya ve Icerik Filtresi'; $s.Save()" >nul 2>&1
if exist "%SHORTCUT_PATH%" (
    echo   [OK] Masaustu kisayolu olusturuldu: OE Watcher
)

echo.
echo [4/4] OE Watcher arka planda baslatiliyor...
start "" "%TARGET_EXE%"

echo.
echo ====================================================================
echo   KURULUM TAMAMLANDI!
echo ====================================================================
echo • Durum: OE Watcher aktif ve calisiyor.
echo • Sistem Tepsisi: Saatin yaninda ikon yerlesti (Sag tik ile yonetebilirsiniz).
echo • Web Kontrol Paneli: http://127.0.0.1:8080 (Sifre: admin)
echo • Kisayol: Masaustunuzdeki 'OE Watcher' ikonundan acabilirsiniz.
echo ====================================================================
echo.
pause
