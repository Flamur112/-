# MuliC2 - TLS-Only Command & Control Framework

A modern, **TLS-encrypted** Command & Control (C2) framework with PowerShell reverse shell generation and secure listener profiles.

**⚠️ IMPORTANT: This system requires TLS certificates and will NOT start without them.**

## ⚡ Quick Start

### Prerequisites
- **Go** (1.21+) - [Download](https://golang.org/dl/)
- **Node.js** (20.19+ or 22.12+) - [Download](https://nodejs.org/)
- **PostgreSQL** - [Download](https://www.postgresql.org/download/) or use Docker
- **OpenSSL** - For generating TLS certificates
- **Git**

### 1. Clone & Setup
```bash
git clone <repository-url>
cd MuliC2
```

### 2. Generate TLS Certificates (REQUIRED)
```powershell
# Generate self-signed certificates for testing
.\generate-certs.ps1

# This creates:
# - ./certs/server.crt (certificate)
# - ./certs/server.key (private key)
```

**⚠️ The server will NOT start without these certificates!**

### 3. Database Setup
```bash
# Create PostgreSQL database
psql -U postgres
CREATE DATABASE mulic2_db;
\q
```

### 4. One-Command Setup (Recommended)
```bash
# Windows
run-mulic2.bat

# Linux/Mac
chmod +x run-mulic2.sh
./run-mulic2.sh
```

**This single script will:**
- Start both backend and frontend servers
- Open the application in your browser
- Everything ready to use!

### 5. Access Your C2 Platform
- **Frontend**: http://localhost:5173
- **Single Page Application** - Login/Register tabs on one page
- **No default credentials** - You must register first!

### 6. Automatic Setup
The launcher scripts now automatically:
- ✅ Detect PostgreSQL installation automatically
- ✅ Ask for pg_ctl.exe path if not found
- ✅ Start PostgreSQL if not running
- ✅ Create the `mulic2_db` database if it doesn't exist
- ✅ Verify database connection before starting services
- ✅ **TLS certificate validation** (server won't start without them)

### 7. Cleanup (When Done)
```bash
# Windows
cleanup-postgres.bat

# Linux/Mac
chmod +x cleanup-postgres.sh
./cleanup-postgres.sh
```

**This will clean up any leftover PostgreSQL, Go, or Node.js processes.**

## 📁 Project Structure

```
MuliC2/
├── backend/           # Go backend server with TLS enforcement
│   ├── handlers/      # HTTP request handlers
│   ├── models/        # Data models
│   ├── services/      # Business logic (TLS listener service)
│   ├── utils/         # Utility functions
│   ├── main.go        # Main server entry point
│   └── config.json    # Backend configuration
├── frontend/          # Vue.js frontend application
│   ├── src/           # Source code
│   ├── public/        # Static assets
│   └── config.json    # Frontend configuration
├── certs/             # TLS certificates (created by generate-certs.ps1)
│   ├── server.crt     # Server certificate
│   └── server.key     # Private key
├── here.ps1           # Core PowerShell reverse shell functions
├── generate-shell.ps1 # PowerShell payload generator
├── generate-certs.ps1 # TLS certificate generator
├── run-mulic2.bat     # Windows launcher script
├── run-mulic2.sh      # Linux/macOS launcher script
├── cleanup-postgres.bat # Windows cleanup script
├── cleanup-postgres.sh  # Linux/macOS cleanup script
└── README.md          # This file
```

## 🔧 Features

### ✅ Implemented
- **TLS Encryption** - All C2 communication encrypted via TLS 1.3/1.2
- **PowerShell Reverse Shell** - Generate and obfuscate payloads
- **User Authentication** - JWT-based login system
- **Listener Profiles** - Multiple named C2 listener configurations
- **Profile Management** - Create, activate, and delete profiles
- **Modern UI** - Vue.js + Element Plus interface
- **Database Integration** - PostgreSQL with proper schemas
- **Security** - Password hashing, session management
- **Port Management** - Automatic conflict detection and validation

### 🚧 In Development
- **Agent Management** - C2 agent registration and control
- **Task Management** - Command execution and monitoring
- **Real-time Communication** - WebSocket-based agent communication
- **Logging & Monitoring** - Comprehensive audit trails

## 🎯 PowerShell Reverse Shell

### Generate Encrypted Payloads
```powershell
# Generate reverse shell for your C2 server
.\generate-shell.ps1 -LHOST "YOUR_SERVER_IP" -LPORT YOUR_SERVER_PORT

# Generate with obfuscation
.\generate-shell.ps1 -LHOST "192.168.1.100" -LPORT 443 -Obfuscate

# Custom output filename
.\generate-shell.ps1 -LHOST "10.0.0.50" -LPORT 8080 -OutputFile "my-shell.ps1"
```

### Features
- **TLS 1.3/1.2 Support** - Modern encryption with automatic fallback
- **Smart Protocol Detection** - Automatically uses best available TLS version
- **Payload Obfuscation** - Base64 encoding and variable randomization
- **Cross-Platform** - Works on Windows 10 2004+ and PowerShell 5.1+

## 🎯 Listener Profiles

The **Listener Profiles** feature allows you to:

1. **Create Multiple Profiles** - Different configurations for different environments
   - Production: `0.0.0.0:8081`
   - Testing: `127.0.0.1:8082`
   - Development: `0.0.0.0:8083`

2. **Profile Management** - Create, activate, and delete profiles
3. **Port Management** - Automatic conflict detection and validation
4. **User Isolation** - Each user manages their own profiles

### Profile Management
- **Create**: Set name, IP, and port
- **Activate**: Switch to a profile
- **Delete**: Remove unused profiles (with agent disconnection warning)

## 🔐 Security

### TLS Encryption (MANDATORY)
- **All C2 traffic encrypted** via TLS 1.3/1.2
- **No plain TCP connections** allowed
- **Certificate validation** enforced
- **Server won't start** without proper certificates

### Application Security
- **Password Hashing** - Secure password storage
- **JWT Tokens** - Secure session management
- **Input Validation** - Comprehensive form validation
- **SQL Injection Protection** - Parameterized queries
- **CORS Configuration** - Proper cross-origin settings

## 🛠️ Development

### Manual Start (Alternative to launcher scripts)
```bash
# Terminal 1: Start Backend
cd backend
go run main.go

# Terminal 2: Start Frontend
cd frontend
npm run dev
```

### Backend Development
```bash
cd backend
go mod download
go run main.go
```

### Frontend Development
```bash
cd frontend
npm install
npm run dev
```

## 📊 API Endpoints

### Authentication
- `POST /api/auth/login` - User login
- `POST /api/auth/register` - User registration
- `POST /api/auth/logout` - User logout

### Profiles
- `POST /api/profile/start` - Start C2 listener with profile
- `POST /api/profile/stop` - Stop C2 listener
- `GET /api/profile/status` - Get listener status

## 🐛 Troubleshooting

### TLS Certificate Issues

**Server Won't Start - Missing Certificates**
```bash
# Error: "certificate file not found: ./certs/server.crt"
# Solution: Generate certificates first
.\generate-certs.ps1
```

**TLS Validation Failed**
```bash
# Ensure certificate files exist and are readable
# Check paths in config.json match actual file locations
```

### Common Issues

**Database Connection Failed**
```bash
# Check PostgreSQL is running
# Windows
netstat -ano | findstr :5432

# Linux/Mac
netstat -tulpn | grep :5432
```

**Port Already in Use**
```bash
# Check what's using the port
# Windows
netstat -ano | findstr :8080

# Linux/Mac
netstat -tulpn | grep :8080
```

**Frontend Build Errors**
```bash
# Clear node modules and reinstall
cd frontend
rm -rf node_modules package-lock.json
npm install
```

## 📝 Configuration

### Port Configuration
Edit `backend/config.json` and `frontend/config.json` to change ports:
- **API Port**: Default 8080 (web interface)
- **C2 Port**: Default 8081 (agent connections)
- **Frontend Port**: Default 5173 (Vue.js dev server)

### Database Configuration
- **Host**: `localhost` (default)
- **Port**: `5432` (default)
- **Database**: `mulic2_db`
- **User**: `postgres` (default)

## 📄 License

This project is for educational and authorized testing purposes only.

## 🆘 Support

If you encounter issues:
1. Check the troubleshooting section above
2. Review the logs in the terminal
3. Ensure all prerequisites are installed
4. Verify database connectivity
5. **Check TLS certificates exist** (most common issue)
6. Ensure OpenSSL is installed and in PATH

---

**Happy Hacking! 🎯**
