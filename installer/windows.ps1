$ErrorActionPreference = "Stop"

# Detect architecture
$arch = ""
if ([System.Environment]::Is64BitOperatingSystem) {
    $envProc = $env:PROCESSOR_ARCHITECTURE
    if ($envProc -eq "ARM64") { $arch = "arm64" }
    else { $arch = "amd64" }
}
else {
    Write-Error "32-bit Windows is not supported. Please use a 64-bit version of Windows."
    exit 1
}

$installDir = "$env:USERPROFILE\.gluax\bin"
$zipName = "gluax-windows-$arch.zip"
$repo = "gluax-lang/gluax"

# Get latest release tag
$latest = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
$url = "https://github.com/$repo/releases/download/$latest/$zipName"

# Create install dir
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Download and extract
$tempZip = "$env:TEMP\$zipName"
Invoke-WebRequest -Uri $url -OutFile $tempZip
Expand-Archive -Path $tempZip -DestinationPath $installDir -Force
Remove-Item $tempZip

# Add to PATH if not present
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$userPath;$installDir", "User")
    Write-Host "Gluax installed. Please restart your terminal or log out/in to update your PATH."
}
else {
    Write-Host "Gluax installed and already in PATH."
}
