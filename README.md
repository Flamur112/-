# MuliC2 - TLS-Only Command & Control Framework

A modern, TLS-encrypted Command & Control (C2) framework with support for multiple concurrent listener profiles, agent management, and unified API communication.

## 🚀 Features

### Core Features
- **TLS-Only Architecture**: All C2 communication encrypted with TLS 1.3/1.2
- **Multiple Listener Profiles**: Support for concurrent C2 listeners on different ports
- **Agent Management**: Complete agent registration, tasking, and result collection
- **Profile-Aware Agent Generation**: Agents built with specific profile configurations
- **Unified API Mode**: Option to serve API through TLS listeners or separate HTTP port
- **Database Persistence**: PostgreSQL backend with profile and agent tracking

### Agent Features
- **Automatic Registration**: Agents register with hostname, OS, architecture, and profile
- **Task Polling**: Configurable polling intervals per profile
- **Command Execution**: PowerShell command execution with result reporting
- **TLS Communication**: All agent-server communication encrypted

### Operator Features
- **Agent Dashboard**: Real-time agent status and management
- **Task Queue**: Command queuing and result viewing
- **Profile Management**: Create and manage C2 listener profiles
- **Multi-Profile Support**: Operate multiple C2 campaigns simultaneously

## 📋 Prerequisites

- **Go 1.21+** (for backend)
- **Node.js 18+** (for frontend)
- **PostgreSQL 12+** (database)
- **TLS Certificates** (required for C2 communication)

## 🔧 Installation

### 1. Clone the Repository
```bash
git clone <repository-url>
cd MuliC2
```

### 2. Generate TLS Certificates (REQUIRED)
```bash
# Using OpenSSL
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=localhost"

# Or use the provided script
./generate-certs.ps1
```

### 3. Automatic Setup
```bash
# Linux
sudo ./run-mulic2.sh

# Windows
.\run-mulic2.bat
```

The setup script will:
- Install and configure PostgreSQL
- Build the backend executable
- Install frontend dependencies
- Start both backend and frontend servers
- Initialize database with default profiles

## 🎯 Usage

### Starting the C2 Server

#### Linux
```bash
sudo ./run-mulic2.sh
```

#### Windows
```powershell
.\run-mulic2.bat
```

### Accessing the Interface
- **Frontend**: http://localhost:5173
- **Backend API**: http://localhost:8080 (or through TLS in unified mode)

### Generating Agents

1. **Navigate to Agents Tab**: In the web interface
2. **Select Profile**: Choose from available C2 profiles
3. **Configure Settings**: 
   - Server Host: Your C2 server IP
   - API Port: Backend API port (8080 by default)
   - Profile ID: Selected profile
   - Poll Interval: How often to check for tasks
4. **Generate Command**: Copy the generated PowerShell command
5. **Deploy Agent**: Run the command on target systems

### Example Agent Command
```powershell
.\Start-MuliAgent.ps1 -ServerHost 192.168.1.100 -ApiPort 8080 -ProfileId default -PollSeconds 5
```

### Tasking Agents

1. **View Agents**: See all registered agents in the dashboard
2. **Queue Tasks**: Select an agent and queue commands
3. **Monitor Results**: View command execution results in real-time

## ⚙️ Configuration

### Profile Configuration

Profiles are defined in `backend/config.json`:

```json
{
  "profiles": [
    {
      "id": "default",
      "name": "Default TLS Profile",
      "projectName": "MuliC2",
      "host": "0.0.0.0",
      "port": 8443,
      "description": "Default TLS-enabled C2 profile",
      "useTLS": true,
      "certFile": "../server.crt",
      "keyFile": "../server.key"
    },
    {
      "id": "production",
      "name": "Production Profile",
      "projectName": "MuliC2",
      "host": "0.0.0.0",
      "port": 443,
      "description": "Production TLS profile on standard HTTPS port",
      "useTLS": true,
      "certFile": "../server.crt",
      "keyFile": "../server.key"
    }
  ]
}
```

### API Communication Modes

#### Separated Mode (Default)
- API runs on HTTP port 8080
- C2 listeners on separate TLS ports
- Configuration: `"api_unified": false`

#### Unified Mode
- API served through TLS listeners
- Single port for both API and C2 communication
- Configuration: `"api_unified": true, "api_unified_port": 8443`

### Database Configuration

```json
{
  "database": {
    "type": "postgres",
    "host": "localhost",
    "port": 5432,
    "user": "postgres",
    "password": "your_password",
    "dbname": "mulic2_db",
    "sslmode": "disable"
  }
}
```

## 🗄️ Database Schema

### Agents Table
```sql
CREATE TABLE agents (
    id SERIAL PRIMARY KEY,
    hostname VARCHAR(255),
    username VARCHAR(255),
    ip VARCHAR(64),
    os VARCHAR(128),
    arch VARCHAR(64),
    profile_id VARCHAR(128),
    status VARCHAR(32) DEFAULT 'online',
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Profiles Table
```sql
CREATE TABLE profiles (
    id VARCHAR(128) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    project_name VARCHAR(255),
    host VARCHAR(64) DEFAULT '0.0.0.0',
    port INTEGER NOT NULL,
    description TEXT,
    use_tls BOOLEAN DEFAULT true,
    cert_file VARCHAR(512),
    key_file VARCHAR(512),
    poll_interval INTEGER DEFAULT 5,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Tasks Table
```sql
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    command TEXT NOT NULL,
    status VARCHAR(32) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Results Table
```sql
CREATE TABLE results (
    task_id INTEGER PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
    output TEXT,
    completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🔒 Security Features

### TLS Encryption
- **TLS 1.3/1.2 Support**: Modern encryption protocols
- **Certificate Validation**: Required for all C2 communication
- **No Plain Text**: All communication encrypted

### Authentication
- **Database Authentication**: Secure user management
- **Session Management**: Protected API endpoints
- **Audit Logging**: Track all operator actions

### Network Security
- **Port Flexibility**: Use any port for C2 listeners
- **Profile Isolation**: Separate configurations per campaign
- **Unified Mode**: Single port for API and C2 (optional)

## 🛠️ API Endpoints

### Agent Endpoints
- `POST /api/agent/register` - Agent registration
- `POST /api/agent/heartbeat` - Agent heartbeat
- `GET /api/agent/tasks` - Fetch pending tasks
- `POST /api/agent/result` - Submit task results

### Operator Endpoints
- `GET /api/agents` - List all agents
- `POST /api/tasks` - Enqueue task for agent
- `GET /api/agent-tasks` - Get agent task history

### Profile Management
- `GET /api/profile/list` - List all profiles
- `POST /api/profile/create` - Create new profile
- `GET /api/profile/get` - Get profile details
- `POST /api/profile/start` - Start profile listener
- `POST /api/profile/stop` - Stop profile listener

## 🚨 Troubleshooting

### Common Issues

#### TLS Certificate Errors
```bash
# Ensure certificates exist
ls -la server.crt server.key

# Generate new certificates if needed
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes
```

#### Port Conflicts
```bash
# Check what's using the port
sudo ss -tuln | grep :8443

# Stop conflicting service or change port in config.json
```

#### Database Connection Issues
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Reset database password
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'your_password';"
```

#### Privileged Port Issues (Linux)
```bash
# For ports < 1024, either:
# 1. Run as root
sudo ./run-mulic2.sh

# 2. Use setcap
sudo setcap 'cap_net_bind_service=+ep' ./backend/mulic2

# 3. Use non-privileged ports (>= 1024)
```

### Logs and Debugging
- **Backend Logs**: Check terminal output for detailed error messages
- **Frontend Logs**: Browser developer console
- **Database Logs**: PostgreSQL logs in `/var/log/postgresql/`

## 📁 Project Structure

```
MuliC2/
├── backend/
│   ├── main.go                 # Main server entry point
│   ├── config.json             # Server configuration
│   ├── handlers/               # API handlers
│   │   ├── agent.go           # Agent API endpoints
│   │   ├── operator.go        # Operator API endpoints
│   │   └── profile.go         # Profile management
│   └── services/
│       └── listener.go        # C2 listener service
├── frontend/
│   ├── src/
│   │   ├── views/
│   │   │   ├── Dashboard.vue  # Main dashboard
│   │   │   └── Terminal.vue   # Agent management
│   │   └── ...
│   └── ...
├── Start-MuliAgent.ps1        # PowerShell agent script
├── run-mulic2.sh              # Linux launcher
├── run-mulic2.bat             # Windows launcher
├── server.crt                 # TLS certificate
├── server.key                 # TLS private key
└── README.md                  # This file
```

## 🔄 Development

### Building from Source
```bash
# Backend
cd backend
go build -o mulic2

# Frontend
cd frontend
npm install
npm run build
```

### Adding New Features
1. **Backend**: Add handlers in `backend/handlers/`
2. **Frontend**: Add components in `frontend/src/views/`
3. **Database**: Update schema in `backend/main.go`
4. **Configuration**: Update `backend/config.json`

## 📄 License

This project is for educational and authorized testing purposes only. Users are responsible for complying with all applicable laws and regulations.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## ⚠️ Disclaimer

This tool is designed for authorized penetration testing and security research. Users must ensure they have proper authorization before using this tool against any systems. The authors are not responsible for any misuse of this software.
