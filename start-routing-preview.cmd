@echo off
setlocal
cd /d "%~dp0frontend"
set "PREVIEW_PNPM=%APPDATA%\npm\pnpm.cmd"
if not exist "%PREVIEW_PNPM%" set "PREVIEW_PNPM=pnpm"
call "%PREVIEW_PNPM%" exec vite --config vite.preview.config.ts --open /routing-preview.html
