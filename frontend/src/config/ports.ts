// Port configuration for MuliC2
// Users can edit config.json to change ports
// NO FALLBACK PORTS - if configured ports don't work, the system will fail

// Load configuration from config.json
let PORT_CONFIG: any = null

// Load configuration from config.json - NO FALLBACKS
const loadPortConfig = async (): Promise<any> => {
  try {
    const response = await fetch('/config.json')
    if (!response.ok) {
      throw new Error(`Failed to load config.json: ${response.status} ${response.statusText}`)
    }
    return await response.json()
  } catch (error) {
    console.error('CRITICAL: Could not load config.json:', error)
    throw new Error('Port configuration file (config.json) could not be loaded. The system cannot start without proper port configuration.')
  }
}

// Initialize port configuration
let PORTS: any = null
let initializationPromise: Promise<void> | null = null

// Initialize ports - this will throw an error if config cannot be loaded
export const initializePorts = async (): Promise<void> => {
  // Return existing promise if already initializing
  if (initializationPromise) {
    return initializationPromise
  }

  // Return immediately if already initialized
  if (PORTS) {
    return
  }

  // Create initialization promise
  initializationPromise = (async () => {
    try {
      PORT_CONFIG = await loadPortConfig()
      PORTS = {
        // Backend API server port (configurable via config.json)
        BACKEND_API: PORT_CONFIG.backend.api_port,
        
        // Default C2 listener port (configurable via config.json)
        C2_DEFAULT: PORT_CONFIG.backend.c2_default_port,
        
        // Frontend development server port
        FRONTEND: PORT_CONFIG.frontend.port
      }
      
      // Validate port configuration
      if (!PORTS.BACKEND_API || !PORTS.C2_DEFAULT) {
        throw new Error('Invalid port configuration: api_port and c2_default_port must be specified')
      }
      
      console.log('Port configuration loaded successfully:', PORTS)
    } catch (error) {
      console.error('Failed to initialize port configuration:', error)
      // Reset promise so it can be retried
      initializationPromise = null
      throw error
    }
  })()

  return initializationPromise
}

// Get ports - throws error if not initialized
export const getPorts = () => {
  if (!PORTS) {
    throw new Error('Ports not initialized. Call initializePorts() first.')
  }
  return PORTS
}

// Helper function to check if a port conflicts with backend API
export const isPortConflicting = (port: number): boolean => {
  if (!PORTS) {
    throw new Error('Ports not initialized. Call initializePorts() first.')
  }
  return port === PORTS.BACKEND_API
}

// Helper function to get available port suggestions (for UI only, not for actual binding)
export const getPortSuggestions = (): number[] => {
  if (!PORTS) {
    throw new Error('Ports not initialized. Call initializePorts() first.')
  }
  
  const suggestions = []
  let port = PORTS.C2_DEFAULT
  
  // Generate 5 port suggestions starting from default
  for (let i = 0; i < 5; i++) {
    if (port !== PORTS.BACKEND_API) {
      suggestions.push(port)
    }
    port++
  }
  
  return suggestions
}
