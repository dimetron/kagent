# Authentication Overview

## UI Authentication flow

### Overview

The UI authentication flow can be configured with several auth strategies, including:

- Anonymous users can access the UI.
- Basic authentication user/password.
- Token based authentication to use Kubernetes service accounts.
- OIDC (OpenID Connect) authentication.

## UI Authentication & Authorization Details

### Current Implementation Status

The UI authentication is currently in **development/demo mode** with minimal security implementation:

#### Frontend (Next.js UI)
- **Hard-coded user**: Uses `admin@kagent.dev` as default user (`ui/src/lib/userStore.ts`)
- **No login/logout UI**: No authentication forms or session management components
- **No route protection**: All routes are publicly accessible
- **No token handling**: API calls use query parameter user identification instead of tokens
- **Local storage only**: User state stored in Zustand store with localStorage persistence

#### Backend (Python FastAPI)
- **Comprehensive auth framework**: Ready for production with multiple providers
- **JWT-based authentication**: Token creation, validation, and expiry management
- **Multiple auth providers**: GitHub OAuth, MSAL, Firebase (configurable)
- **WebSocket authentication**: Support for real-time connection auth
- **Middleware-based protection**: Request filtering and authorization enforcement

### Supported Authentication Strategies

#### 1. Anonymous Access (Current Default)
```typescript
// ui/src/lib/userStore.ts
const defaultUser = { userId: "admin@kagent.dev" }
```

#### 2. OAuth Providers (Backend Ready)
- **GitHub OAuth**: Complete implementation with client/secret configuration
- **Microsoft MSAL**: Framework ready, needs configuration
- **Firebase Auth**: Framework ready, needs configuration

#### 3. Token-Based Authentication
- **JWT tokens**: Backend creates and validates tokens with configurable expiry
- **Bearer authentication**: API endpoints support Authorization header validation
- **Service account tokens**: Kubernetes token integration (planned)

### Authorization Architecture

#### Current State
- **No RBAC implementation**: All users have identical permissions
- **No route guards**: Frontend has no protected routes
- **No feature flags**: All UI components visible to all users
- **Hard-coded permissions**: Backend user model includes roles but unused

#### Planned Authorization
- **Role-based access control**: Admin, User, ReadOnly roles
- **Feature-based permissions**: Agent creation, team management, tool access
- **Resource-level authorization**: User-specific agent and team visibility
- **Kubernetes RBAC integration**: Service account permission mapping

### API Authentication Integration

#### Current API Pattern
```typescript
// All API calls append hard-coded user
const response = fetch(`/api/agents?user_id=admin@kagent.dev`)
```

#### Production API Pattern
```typescript
// Token-based authentication
const response = fetch('/api/agents', {
  headers: {
    'Authorization': `Bearer ${jwt_token}`,
    'Content-Type': 'application/json'
  }
})
```

### Authentication Flow Details

#### Development Flow (Current)
```mermaid
graph LR
    A[User] --> B[UI Direct Access]
    B --> C[Hard-coded admin@kagent.dev]
    C --> D[API calls with user_id param]
    D --> E[Backend bypasses auth for excluded paths]
```

#### Production Flow (Target)
```mermaid
graph LR
    A[User] --> B[Login UI]
    B --> C[OAuth Provider]
    C --> D[JWT Token]
    D --> E[Protected UI Routes]
    E --> F[Authenticated API calls]
    F --> G[Backend token validation]
    G --> H[Authorized response]
```

### Session Management

#### Frontend Session (Current)
- **Storage**: Zustand store with localStorage persistence
- **Duration**: Indefinite (no expiration)
- **Security**: No token encryption or secure storage
- **State**: Only stores userId string

#### Backend Session (Available)
- **JWT tokens**: Configurable expiry (default 60 minutes)
- **Token validation**: Signature and expiry checking
- **WebSocket sessions**: Real-time connection authentication
- **Refresh tokens**: Framework ready for implementation

### Security Considerations

#### Current Security Gaps
1. **No authentication barriers**: UI completely bypasses security
2. **Hard-coded credentials**: Development-only user configuration
3. **No session expiry**: Users never logged out automatically
4. **Client-side only auth**: No server-side session validation
5. **No audit logging**: User actions not tracked for security

#### Production Security Requirements
1. **OAuth integration**: Proper third-party authentication flow
2. **JWT token management**: Secure token storage and refresh
3. **Route protection**: Authentication guards for sensitive pages
4. **Role-based rendering**: UI components based on user permissions
5. **Session security**: Automatic logout and token expiry handling
6. **CSRF protection**: Cross-site request forgery prevention
7. **Audit logging**: User action tracking and security monitoring

### Environment Configuration

#### Backend Auth Environment Variables
```bash
# Auth provider selection
AUTOGENSTUDIO_AUTH_TYPE=none|github|msal|firebase

# JWT configuration
AUTOGENSTUDIO_JWT_SECRET=your-secret-key
AUTOGENSTUDIO_TOKEN_EXPIRY=3600

# GitHub OAuth
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret

# Microsoft MSAL
MSAL_CLIENT_ID=your-msal-client-id
MSAL_CLIENT_SECRET=your-msal-client-secret
MSAL_TENANT_ID=your-tenant-id

# Firebase
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_API_KEY=your-api-key
```

#### Frontend Configuration (Missing)
```typescript
// Needed in ui/.env.local
NEXT_PUBLIC_AUTH_PROVIDER=github|msal|firebase
NEXT_PUBLIC_API_BASE_URL=https://api.kagent.dev
NEXT_PUBLIC_OAUTH_REDIRECT_URI=https://ui.kagent.dev/auth/callback
```

## Tools Authentication

### Overview

Tools authentication in Kagent leverages the **Model Context Protocol (MCP)** authentication framework. MCP provides a standardized approach for securing tool integrations using OAuth 2.1 and OpenID Connect standards.

**Reference**: [MCP Authentication Documentation](https://mcp-auth.dev/docs)

### MCP Authentication Architecture

MCP authentication follows a **delegated authentication model** where:
- Tools act as **resource servers**
- Authentication is delegated to **compatible OAuth providers**
- **Minimal server-side authentication logic** is required
- **Standards-compliant** OAuth 2.1 and OpenID Connect implementation

### Authentication Methods

#### 1. Bearer Token Authentication
```python
# MCP Auth configuration
mcp_auth = MCPAuth(
    server=fetch_server_config(
        '<auth-server-url>',
        type=AuthServerType.OIDC
    )
)

# Bearer authentication middleware
bearer_auth = mcp_auth.bearer_auth_middleware(
    "jwt", 
    required_scopes=["read", "write"]
)
```

#### 2. JWT Token Validation
- **Standards-compliant JWT validation**
- **Scope-based access control**
- **Metadata endpoint configuration**
- **Provider-delegated token verification**

### Tool Server Authentication Flow

```mermaid
graph LR
    A[Agent] --> B[Tool Request]
    B --> C[MCP Auth Middleware]
    C --> D[JWT Token Validation]
    D --> E[Scope Verification]
    E --> F[OAuth Provider]
    F --> G[Token Valid]
    G --> H[Tool Execution]
    H --> I[Response]
```

### Integration with Kagent Tools

#### Built-in Tool Server (`go/tools/`)
- **MCP protocol integration**: Uses `github.com/mark3labs/mcp-go` library
- **Bearer token support**: JWT-based authentication for tool requests
- **Scope-based authorization**: Different permission levels for tool operations
- **Provider-agnostic**: Compatible with various OAuth providers

#### Tool Authentication Configuration
```go
// go/tools/ configuration
type ToolServerConfig struct {
    Auth struct {
        Provider     string   `yaml:"provider"`     // OAuth provider
        ServerURL    string   `yaml:"serverUrl"`    // Auth server metadata URL
        RequiredScopes []string `yaml:"scopes"`     // Required OAuth scopes
        JWTValidation struct {
            Issuer   string `yaml:"issuer"`
            Audience string `yaml:"audience"`
        } `yaml:"jwtValidation"`
    } `yaml:"auth"`
}
```

### Security Considerations

#### MCP Security Best Practices
1. **Standards Compliance**: Use OAuth 2.1 and OpenID Connect compliant providers
2. **Minimal Authentication Logic**: Delegate full authentication to the provider
3. **Scope-based Access**: Implement granular permission control
4. **Token Validation**: Verify JWT signatures and required scopes
5. **Metadata Endpoints**: Use provider metadata for configuration

#### Tool-Specific Security
- **Tool isolation**: Each tool server can have independent authentication
- **Principle of least privilege**: Tools request only necessary scopes
- **Audit logging**: Tool access and operations are logged for security
- **Resource protection**: Tools validate user permissions before execution

### Environment Configuration

#### MCP Authentication Variables
```bash
# MCP Auth Server Configuration
MCP_AUTH_SERVER_URL=https://auth.example.com
MCP_AUTH_PROVIDER=oidc

# JWT Configuration
MCP_JWT_ISSUER=https://auth.example.com
MCP_JWT_AUDIENCE=kagent-tools

# Required Scopes
MCP_REQUIRED_SCOPES=tools:read,tools:execute,k8s:read
```

#### Tool Server Configuration
```yaml
# helm/kagent/values.yaml
toolServer:
  auth:
    enabled: true
    provider: "oidc"
    serverUrl: "https://auth.example.com"
    scopes:
      - "tools:read"
      - "tools:execute"
      - "k8s:read"
    jwtValidation:
      issuer: "https://auth.example.com"
      audience: "kagent-tools"
```

### Future Roadmap

- **MCP Specification Evolution**: Transitioning to pure resource server role
- **Enhanced Provider Support**: Additional OAuth provider integrations  
- **Fine-grained Permissions**: Tool-level and operation-level authorization
- **Cross-tool Authentication**: Shared authentication context across tool servers

