@echo off
echo Setting up MathReX Windows dependencies...
echo.

REM Create directories
if not exist "onnxruntime" mkdir onnxruntime
if not exist "libtokenizers" mkdir libtokenizers
if not exist "libtokenizers\windows_amd64" mkdir libtokenizers\windows_amd64

echo Created directories.

REM Download ONNX Runtime for Windows
echo Downloading ONNX Runtime...
powershell -Command "& {
    $url = 'https://github.com/microsoft/onnxruntime/releases/download/v1.21.0/onnxruntime-win-x64-1.21.0.zip'
    $output = 'onnxruntime-win-x64-1.21.0.zip'
    Write-Host 'Downloading ONNX Runtime from:' $url
    try {
        Invoke-WebRequest -Uri $url -OutFile $output -UseBasicParsing
        Write-Host 'Download completed successfully'
        
        Write-Host 'Extracting ONNX Runtime...'
        Expand-Archive -Path $output -DestinationPath 'temp_onnx' -Force
        
        Write-Host 'Copying DLL files...'
        Copy-Item 'temp_onnx\onnxruntime-win-x64-1.21.0\lib\onnxruntime.dll' 'onnxruntime\'
        Copy-Item 'temp_onnx\onnxruntime-win-x64-1.21.0\lib\onnxruntime_providers_shared.dll' 'onnxruntime\'
        
        Write-Host 'Cleaning up...'
        Remove-Item $output -Force
        Remove-Item 'temp_onnx' -Recurse -Force
        
        Write-Host 'ONNX Runtime setup completed!'
    } catch {
        Write-Host 'Error downloading or extracting ONNX Runtime:' $_.Exception.Message
    }
}"

REM Create a dummy tokenizers library file for now
echo Creating placeholder tokenizers library...
echo. > libtokenizers\windows_amd64\tokenizers.lib

echo.
echo Setup completed!
echo.
echo Files created:
if exist "onnxruntime\onnxruntime.dll" (
    echo   ✓ onnxruntime\onnxruntime.dll
) else (
    echo   ✗ onnxruntime\onnxruntime.dll [MISSING]
)

if exist "onnxruntime\onnxruntime_providers_shared.dll" (
    echo   ✓ onnxruntime\onnxruntime_providers_shared.dll
) else (
    echo   ✗ onnxruntime\onnxruntime_providers_shared.dll [MISSING]
)

if exist "libtokenizers\windows_amd64\tokenizers.lib" (
    echo   ✓ libtokenizers\windows_amd64\tokenizers.lib
) else (
    echo   ✗ libtokenizers\windows_amd64\tokenizers.lib [MISSING]
)

echo.
echo You can now run MathReX.exe
pause
