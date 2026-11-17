@echo off
cd /d "%~dp0"
if not exist "%~dp0todoweb.exe" (
    echo L'executable todoweb.exe est absent. Tentative de reconstruction...
    where go >nul 2>&1
    if errorlevel 1 (
        echo Go n'est pas installe. Installez Go puis relancez ce fichier.
        pause
        exit /b 1
    )
    go build -o todoweb.exe .
    if errorlevel 1 (
        echo Echec de la compilation de todoweb.exe.
        pause
        exit /b 1
    )
)
echo Demarrage de Todo Web...
echo Laissez cette fenetre ouverte pendant l'utilisation.
echo.
"%~dp0todoweb.exe"
echo.
echo L'application s'est arretee. Si un message d'erreur apparait ci-dessus, prenez-le en photo.
pause
