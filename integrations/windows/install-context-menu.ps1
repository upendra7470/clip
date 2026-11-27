$ErrorActionPreference = "Stop"

$clip = (Get-Command clip -ErrorAction Stop).Source
$fileKey = "HKCU:\Software\Classes\*\shell\ClipSmart"
$directoryKey = "HKCU:\Software\Classes\Directory\shell\ClipSmart"

New-Item -Path $fileKey -Force | Out-Null
Set-Item -Path $fileKey -Value "Copy text with Clip"
New-Item -Path "$fileKey\command" -Force | Out-Null
Set-Item -Path "$fileKey\command" -Value ('"{0}" --smart "%1"' -f $clip)

New-Item -Path $directoryKey -Force | Out-Null
Set-Item -Path $directoryKey -Value "Copy documents with Clip"
New-Item -Path "$directoryKey\command" -Force | Out-Null
Set-Item -Path "$directoryKey\command" -Value ('"{0}" --smart --recursive "%1"' -f $clip)

Write-Host "Clip context-menu actions installed for the current user."
