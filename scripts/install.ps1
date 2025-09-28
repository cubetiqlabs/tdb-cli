param(
    [string]$Version = "latest"
)
$repo = "cubetiqlabs/tinydb"
$name = "tdb"
function Get-LatestTag {
    $uri = "https://api.github.com/repos/$repo/releases/latest"
    $response = Invoke-RestMethod -Uri $uri -UseBasicParsing
    return $response.tag_name.TrimStart("v")
}
if ($Version -eq "latest") {
    $Version = Get-LatestTag
}
$arch = $env:PROCESSOR_ARCHITECTURE.ToLower()
switch ($arch) {
    "amd64" { $arch = "amd64" }
    "x86" { $arch = "386" }
    "arm64" { $arch = "arm64" }
}
$asset = "${name}_windows_${arch}.zip"
$uri = "https://github.com/$repo/releases/download/v$Version/$asset"
$temp = New-Item -ItemType Directory -Path ([System.IO.Path]::GetTempPath()) -Name ([System.Guid]::NewGuid().ToString())
$zipPath = Join-Path $temp $asset
Write-Host "Downloading $uri..."
Invoke-WebRequest -Uri $uri -OutFile $zipPath
Expand-Archive -Path $zipPath -DestinationPath $temp -Force
$binary = Get-ChildItem -Path $temp -Filter "tdb.exe" -Recurse | Select-Object -First 1
if (-not $binary) {
    Write-Error "Downloaded archive did not contain tdb.exe"
    exit 1
}
$installDir = $env:TDB_INSTALL_DIR
if ([string]::IsNullOrWhiteSpace($installDir)) {
    $installDir = Join-Path $env:ProgramFiles "TinyDB"
}
$dest = Join-Path $installDir "tdb.exe"
New-Item -ItemType Directory -Path (Split-Path $dest) -Force | Out-Null
Move-Item -Path $binary.FullName -Destination $dest -Force
Write-Host "Installed tdb to $dest"
