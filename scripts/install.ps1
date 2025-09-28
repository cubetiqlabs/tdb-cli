param(
    [string]$Version = "latest"
)
$repo = "cubetiqlabs/tdb-cli"
$name = "tdb"

function Get-LatestTag {
    $uri = "https://api.github.com/repos/$repo/releases/latest"
    $response = Invoke-RestMethod -Uri $uri -UseBasicParsing
    return $response.tag_name
}

if ($Version -eq "latest") {
    $tag = Get-LatestTag
    if (-not $tag) {
        throw "Unable to determine latest release tag."
    }
    $Version = $tag.TrimStart("v")
} else {
    if ($Version.StartsWith("v")) {
        $tag = $Version
        $Version = $Version.TrimStart("v")
    } else {
        $tag = "v$Version"
    }
}

if (-not $tag) {
    $tag = "v$Version"
}

$arch = $env:PROCESSOR_ARCHITECTURE.ToLower()
switch ($arch) {
    "amd64" { $arch = "amd64" }
    "x86" { $arch = "386" }
    "arm64" { $arch = "arm64" }
}
$asset = "${name}_windows_${arch}.zip"
$uri = "https://github.com/$repo/releases/download/$tag/$asset"
$checksumUri = "https://github.com/$repo/releases/download/$tag/SHA256SUMS"
$temp = New-Item -ItemType Directory -Path ([System.IO.Path]::GetTempPath()) -Name ([System.Guid]::NewGuid().ToString())
$zipPath = Join-Path $temp $asset
Write-Host "Downloading $uri..."
Invoke-WebRequest -Uri $uri -OutFile $zipPath

$checksumPath = Join-Path $temp "SHA256SUMS"
Invoke-WebRequest -Uri $checksumUri -OutFile $checksumPath

$expected = Select-String -Path $checksumPath -Pattern " $asset" | ForEach-Object {
    $_.Line.Split(' ', [System.StringSplitOptions]::RemoveEmptyEntries)[0]
} | Select-Object -First 1

if (-not $expected) {
    throw "Checksum entry for $asset not found."
}

$actual = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash.ToLower()
if ($actual.ToLower() -ne $expected.ToLower()) {
    throw "Checksum mismatch for $asset. Expected $expected but got $actual."
}

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

Remove-Item -Recurse -Force $temp
