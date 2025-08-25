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

### 2. TLS Certificates (REQUIRED)
Provide `server.crt` and `server.key` at the repo root. The server will refuse to start without them.
You may create them yourself with OpenSSL or your PKI.

### 3. Database Setup
No manual steps needed. The launcher script configures PostgreSQL, sets a password, and creates the database if missing.

### 4. One-Command Setup (Recommended)
```bash
# Windows
run-mulic2.bat

# Linux/Mac
chmod +x run-mulic2.sh
./run-mulic2.sh
```

**This single script will:**
- Start PostgreSQL if required and fix common Linux issues
- Ensure DB user/password match config
- Validate TLS certs
- Start backend and frontend

### 5. Access Your C2 Platform
- **Frontend**: http://localhost:5173
- **Single Page Application** - Login/Register tabs on one page
- **No default credentials** - You must register first!

### 6. Automatic Setup
The launcher scripts automatically:
- ✅ Start PostgreSQL (Linux) and fix peer/md5 auth
- ✅ Create `mulic2_db` if needed
- ✅ Apply DB password to match backend config
- ✅ Validate TLS (server won’t start without certs)

### 7. SPA
The frontend is a Single Page Application (SPA). Use the UI to create/start listener profiles. Errors appear in the page and in the terminal.

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
├── server.crt         # TLS certificate (provide yourself)
├── server.key         # TLS private key (provide yourself)
├── here.ps1           # Core PowerShell reverse shell functions
├── here.ps1           # Polymorphic PowerShell reverse shell generator
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
Use `here.ps1` directly. It generates a TLS 1.3/1.2 reverse shell, obfuscated and wrapped for execution.

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
Edit `backend/config.json` profiles to set listener host/port.
- To use 443 on Linux without root, the launcher applies `setcap cap_net_bind_service=+ep` to the backend binary.
- If 443 is in use, free it (nginx/apache) or select another port.

### Database Configuration
Set in `backend/config.json` under `database`.
- To change the password, either edit the file then re-run the launcher, or run:
  `sudo -u postgres psql -h /var/run/postgresql -d postgres -c "ALTER USER postgres PASSWORD 'NEW_PASSWORD';"`

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
