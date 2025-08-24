# MulisC2 - Command & Control Center

A stealthy, terminal-style Command & Control (C2) server frontend built with Vue.js 3, featuring a dark theme optimized for operational security.

## 🎨 Theme & Design

**Stealthy Black & Red Theme:**
- **Primary Background**: Pure black (#000000) for maximum stealth
- **Secondary Elements**: Dark grays (#0a0a0a, #1a1a1a, #2a2a2a)
- **Accent Color**: Bright red (#ff0000) for critical elements and highlights
- **Typography**: Monospace font (Courier New) for authentic terminal feel
- **Visual Effects**: Subtle red glows, grid patterns, and terminal-style borders

## 🚀 Single Page Application (SPA) Features

**Real-time Performance:**
- All views (Login, Dashboard, Agents, Tasks, Logs, Settings) load on one page
- Navigation happens without full page reloads
- Instant switching between modules
- Optimized for real-time updates (implant status, VNC streams, task monitoring)

**Key Benefits:**
- ⚡ **Fast Response**: No page refresh delays
- 🔄 **Real-time Updates**: Live implant status and task progress
- 📱 **Responsive**: Works seamlessly on all devices
- 🛡️ **Stealth**: Minimal visual footprint, terminal aesthetic
- 🔒 **Secure**: Clean, professional interface for operational use

## 🛠️ Technology Stack

- **Frontend**: Vue.js 3 + TypeScript
- **UI Framework**: Element Plus (customized for stealth theme)
- **Styling**: SCSS with CSS custom properties
- **Routing**: Vue Router 4 (SPA navigation)
- **State Management**: Pinia
- **Build Tool**: Vite
- **Charts**: ECharts for data visualization

## 📁 Project Structure

```
src/
├── views/           # Main application views
│   ├── Login.vue    # Authentication interface
│   ├── Dashboard.vue # Main control panel
│   ├── Agents.vue   # Implant management
│   ├── Tasks.vue    # Task assignment & monitoring
│   ├── Logs.vue     # System logs & audit trail
│   └── Settings.vue # Configuration & preferences
├── layout/          # Application layout components
│   └── Layout.vue   # Main navigation & structure
├── router/          # SPA routing configuration
├── styles/          # Global styles & theme variables
└── main.ts          # Application entry point
```

## 🎯 Key Features

### Authentication
- Secure login with username/password
- Remember me functionality
- Session management

### Dashboard
- Real-time agent statistics
- Live activity monitoring
- Quick action buttons
- Interactive charts and graphs

### Agent Management
- Implant status monitoring
- Connection management
- Real-time health checks

### Task System
- Task creation and assignment
- Progress tracking
- Result collection
- Status monitoring

### Logging & Monitoring
- Comprehensive audit trail
- Real-time log streaming
- Search and filtering
- Export capabilities

## 🚀 Getting Started

1. **Install Dependencies:**
   ```bash
   npm install
   ```

2. **Start Development Server:**
   ```bash
   npm run dev
   ```

3. **Build for Production:**
   ```bash
   npm run build
   ```

## 🎨 Customization

The stealth theme uses CSS custom properties for easy customization:

```scss
:root {
  --primary-black: #000000;      // Main background
  --accent-red: #ff0000;         // Primary accent
  --text-white: #ffffff;         // Primary text
  --border-color: #333333;       // Borders
}
```

## 🔒 Security Features

- **Minimal Attack Surface**: Clean, simple interface
- **Stealth Design**: Terminal aesthetic reduces visual detection
- **Responsive Layout**: Works on any device size
- **Professional Appearance**: Suitable for enterprise environments

## 📱 Browser Support

- Chrome/Chromium (recommended)
- Firefox
- Safari
- Edge

## 🤝 Contributing

This is a professional C2 server interface. Please ensure all contributions maintain the stealth aesthetic and operational security requirements.

## 📄 License

Proprietary - Command & Control Operations Use Only

---

**Note**: This interface is designed for legitimate security operations and penetration testing. Ensure compliance with all applicable laws and regulations.
