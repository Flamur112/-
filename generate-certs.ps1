# MuliC2 Certificate Generator
# This script generates self-signed certificates for TLS testing

param(
    [string]$CertDir = "./certs",
    [string]$CertName = "server",
    [int]$Days = 365
)

Write-Host "[*] MuliC2 Certificate Generator" -ForegroundColor Green
Write-Host "[*] Generating self-signed certificates for TLS testing" -ForegroundColor Yellow

# Check if OpenSSL is available
$opensslPath = $null

# Try to find OpenSSL in common locations
$possiblePaths = @(
    "C:\Program Files\OpenSSL-Win64\bin\openssl.exe",
    "C:\Program Files\Git\mingw64\bin\openssl.exe",
    "C:\Program Files\Git\usr\bin\openssl.exe"
)

foreach ($path in $possiblePaths) {
    if (Test-Path $path) {
        $opensslPath = $path
        break
    }
}

# Try PATH as fallback
if (-not $opensslPath) {
    try {
        $opensslVersion = openssl version 2>$null
        if ($LASTEXITCODE -eq 0) {
            $opensslPath = "openssl"
        }
    } catch {
        # OpenSSL not in PATH
    }
}

if (-not $opensslPath) {
    Write-Host "[-] Error: OpenSSL not found in common locations or PATH" -ForegroundColor Red
    Write-Host "[*] Please install OpenSSL or ensure it's accessible" -ForegroundColor Yellow
    exit 1
}

Write-Host "[+] OpenSSL found: $opensslPath" -ForegroundColor Green

# Create certificates directory
if (-not (Test-Path $CertDir)) {
    Write-Host "[*] Creating certificates directory: $CertDir" -ForegroundColor Cyan
    New-Item -ItemType Directory -Path $CertDir -Force | Out-Null
}

try {
    # Generate private key
    Write-Host "[*] Generating private key..." -ForegroundColor Cyan
    $keyFile = Join-Path $CertDir "$CertName.key"
    & $opensslPath genrsa -out $keyFile 4096
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to generate private key"
    }
    Write-Host "[+] Private key generated: $keyFile" -ForegroundColor Green

    # Generate certificate signing request
    Write-Host "[*] Generating certificate signing request..." -ForegroundColor Cyan
    $csrFile = Join-Path $CertDir "$CertName.csr"
    & $opensslPath req -new -key $keyFile -out $csrFile -subj "/CN=localhost/O=MuliC2/OU=Testing/C=US"
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to generate CSR"
    }
    Write-Host "[+] CSR generated: $csrFile" -ForegroundColor Green

    # Generate self-signed certificate
    Write-Host "[*] Generating self-signed certificate..." -ForegroundColor Cyan
    $certFile = Join-Path $CertDir "$CertName.crt"
    & $opensslPath x509 -req -in $csrFile -signkey $keyFile -out $certFile -days $Days
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to generate certificate"
    }
    Write-Host "[+] Certificate generated: $certFile" -ForegroundColor Green

    # Set proper permissions (Windows)
    Write-Host "[*] Setting file permissions..." -ForegroundColor Cyan
    $acl = Get-Acl $keyFile
    $acl.SetAccessRuleProtection($true, $false)
    $rule = New-Object System.Security.AccessControl.FileSystemAccessRule("Administrators","FullControl","Allow")
    $acl.AddAccessRule($rule)
    Set-Acl $keyFile $acl

    # Clean up CSR file (not needed for TLS)
    Remove-Item $csrFile -Force
    Write-Host "[+] Cleaned up temporary files" -ForegroundColor Green

    # Display certificate info
    Write-Host "`n[*] Certificate Information:" -ForegroundColor Cyan
    & $opensslPath x509 -in $certFile -text -noout | Select-String -Pattern "Subject:|Issuer:|Not Before:|Not After:|DNS:"
    
    # Display file sizes
    $keySize = (Get-Item $keyFile).Length
    $certSize = (Get-Item $certFile).Length
    
    Write-Host "`n[*] Generated Files:" -ForegroundColor Cyan
    Write-Host "  - Private Key: $keyFile ($keySize bytes)" -ForegroundColor Gray
    Write-Host "  - Certificate: $certFile ($certSize bytes)" -ForegroundColor Gray
    Write-Host "  - Directory: $CertDir" -ForegroundColor Gray

    Write-Host "`n[+] Certificate generation completed successfully!" -ForegroundColor Green
    Write-Host "[*] You can now start your MuliC2 server with TLS enabled" -ForegroundColor Yellow
    Write-Host "[*] The server will automatically use these certificates" -ForegroundColor Yellow

} catch {
    Write-Host "[-] Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "[*] Certificate generation failed" -ForegroundColor Yellow
    exit 1
}

Write-Host "`n[*] Next steps:" -ForegroundColor Cyan
Write-Host "  1. Ensure your config.json has the correct paths:" -ForegroundColor Gray
Write-Host "     - certFile: '$CertDir/$certFile'" -ForegroundColor Gray
Write-Host "     - keyFile: '$CertDir/$keyFile'" -ForegroundColor Gray
Write-Host "  2. Start your MuliC2 server" -ForegroundColor Gray
Write-Host "  3. The server will now use TLS encryption" -ForegroundColor Gray
