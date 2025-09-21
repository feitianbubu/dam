# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **云一Dam** (Cloud Dam) - an AI-powered image and video data management system based on the gin-vue-admin framework (version 2.8.5). It's a full-stack web application that provides comprehensive RBAC (Role-Based Access Control), file management, and media processing capabilities.

**Architecture**: Full-stack separation with Go backend (Gin framework) and Vue.js frontend.

## Development Environment Requirements

- **Go**: 1.23+ (currently using go1.23.9)
- **Node.js**: 20+
- **Database**: MySQL 5.7+, PostgreSQL, SQLite, SQL Server, or MongoDB
- **Redis**: For caching and session management
- **IDE**: Recommended Goland for Go development

## Common Development Commands

### Backend (server/)
```bash
# Install dependencies and generate code
go generate

# Build backend
go build -v

# Run development server
go run .

# Generate Swagger docs
swag init

# Generate specialized Swagger docs for Dam
swag init -t Dam -o docs/oas --instanceName swag

# Run tests
go test ./...

# Tidy dependencies
go mod tidy
```

### Frontend (web/)
```bash
# Install dependencies
npm install

# Development server with hot reload
npm run serve

# Production build
npm run build

# Preview production build
npm run preview
```

### Docker & Build
```bash
# Build both frontend and backend
make build

# Build web only
make build-web

# Build server only
make build-server

# Local build (no Docker)
make build-local

# Build Docker images
make image

# Package plugin (replace email with plugin name)
make plugin PLUGIN=email
```

## High-Level Architecture

### Backend Architecture (server/)

The backend follows a strict **layered architecture** pattern:

1. **Model Layer** (`model/`): Database entities and request/response structures
   - Data models inherit from `global.GVA_MODEL` (includes ID, CreatedAt, UpdatedAt)
   - Request models in `model/request/` for API parameter binding
   - Response models handle API output formatting

2. **Service Layer** (`service/`): Core business logic and database operations
   - All business logic resides here
   - No HTTP-specific code (gin.Context) allowed
   - Functions return data and error objects

3. **API Layer** (`api/`): HTTP request handlers
   - Validates input parameters
   - Calls service layer functions
   - Returns standardized JSON responses using `response` package
   - **Must include complete Swagger annotations for all endpoints**

4. **Router Layer** (`router/`): Route definitions and middleware configuration
   - Maps HTTP paths to API handlers
   - Configures authentication and authorization middleware

5. **Initialize Layer** (`initialize/`): System initialization and plugin loading
   - Database table migration (`gorm.go`)
   - Route registration (`router.go`)
   - Menu initialization (`menu.go`)
   - Configuration loading (`viper.go`)

### Module Organization Pattern

Each functional module follows the `enter.go` pattern:

- `api/enter.go`: Exports `ApiGroup` with all API handlers
- `service/enter.go`: Exports `ServiceGroup` with all services
- `router/enter.go`: Exports `RouterGroup` with all routers

This prevents circular dependencies and provides clean module boundaries.

### Frontend Architecture (web/)

Built with **Vue 3 + Composition API**:

1. **API Layer** (`src/api/`): HTTP client abstractions using axios
2. **Components** (`src/components/`): Reusable UI components
3. **Pages** (`src/view/`): Business page components
4. **State Management** (`src/pinia/`): Pinia stores for global state
5. **Router** (`src/router/`): Vue Router configuration with permission guards
6. **Utils** (`src/utils/`): Helper functions and utilities

### Key Technology Stack

**Backend:**
- **Framework**: Gin 1.10.0 (Go web framework)
- **ORM**: GORM 1.25.12 (database abstraction)
- **Auth**: JWT 5.2.2 + Casbin 2.103.0 (authentication + authorization)
- **Config**: Viper 1.19.0 (configuration management)
- **Logging**: Zap 1.27.0 (structured logging)
- **Cache**: Redis 9.7.0
- **Documentation**: Swagger/OpenAPI with swaggo

**Frontend:**
- **Framework**: Vue 3.5.7 with Composition API
- **Build Tool**: Vite 6.2.3
- **UI Library**: Element Plus 2.10.2
- **State Management**: Pinia 2.2.2
- **CSS Framework**: UnoCSS 66.4.2 (atomic CSS)
- **Router**: Vue Router 4.4.3
- **HTTP Client**: Axios 1.8.2
- **Charts**: ECharts 5.5.1

### Plugin System

The system supports plugins for extending functionality:

**Backend Plugin Structure:**
```
server/plugin/[plugin-name]/
├── api/           # API handlers
├── model/         # Data models
├── service/       # Business logic
├── router/        # Route definitions
├── initialize/    # Initialization code
└── plugin.go      # Plugin entry point
```

**Frontend Plugin Structure:**
```
web/src/plugin/[plugin-name]/
├── api/           # API calls
├── components/    # UI components
├── view/          # Pages
└── config.js      # Plugin configuration
```

### Configuration System

- **Main config**: `server/config.yaml`
- **Environment-specific**: `web/.env.development`, `web/.env.production`
- Supports multiple databases, cloud storage providers, and caching options
- MCP (Model Context Protocol) integration for AI-enhanced development

### Key Features

- **RBAC Permission System**: Role-based access control with Casbin
- **Multi-database Support**: MySQL, PostgreSQL, SQLite, SQL Server, MongoDB
- **Cloud Storage Integration**: Aliyun OSS, AWS S3, MinIO, Qiniu, Tencent COS
- **Code Generation**: Automated CRUD code generation
- **File Management**: Upload, download, and media processing
- **Swagger Documentation**: Auto-generated API documentation
- **Plugin Architecture**: Extensible plugin system
- **MCP Support**: AI-enhanced development capabilities

### Database Design

Tables follow the GVA framework conventions:
- All models inherit base fields (ID, CreatedAt, UpdatedAt, DeletedAt)
- Use GORM tags for database mapping
- Soft delete enabled by default
- Support for multiple database engines

### Security Features

- JWT-based authentication
- Casbin authorization with policy management
- CORS configuration
- Request rate limiting
- Input validation and sanitization
- File upload restrictions

### Development Workflow

1. **Backend Development**: Model → Service → API → Router → Initialize
2. **Frontend Development**: API → Components → Pages → Routes
3. **Plugin Development**: Follow the plugin structure for both backend and frontend
4. **Testing**: Use Go's built-in testing framework and frontend testing tools
5. **Documentation**: Update Swagger annotations for API changes

This architecture ensures maintainability, scalability, and clear separation of concerns across the full-stack application.