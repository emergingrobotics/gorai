# NATS Authentication for Gorai

**Version:** 1.0
**Status:** Specification
**Last Updated:** 2026-02-06

## Table of Contents

1. [Overview](#overview)
2. [Authentication Methods](#authentication-methods)
   - [Token Authentication](#1-token-authentication)
   - [User/Password Authentication](#2-userpassword-authentication)
   - [NKeys Authentication](#3-nkeys-authentication)
   - [JWT/Accounts](#4-jwtaccounts-decentralized-authentication)
   - [TLS Client Certificates](#5-tls-client-certificates-mtls)
3. [Comparison Matrix](#comparison-matrix)
4. [Recommendations for Gorai](#recommendations-for-gorai)
5. [Implementation Guide](#implementation-guide)
6. [RDL Configuration](#rdl-configuration)
7. [Server Configuration](#server-configuration)
8. [Credential Management](#credential-management)
9. [Security Best Practices](#security-best-practices)
10. [Troubleshooting](#troubleshooting)

---

## Overview

Gorai uses NATS as its messaging backbone for all inter-component communication. In distributed deployments where components run on multiple devices (e.g., main robot, ground station, remote sensors), securing NATS connections becomes critical.

### Why Authentication Matters

Without authentication, any device on the network can:
- Subscribe to all robot telemetry (privacy/security risk)
- Publish commands to actuators (safety risk)
- Access JetStream data (data integrity risk)
- Impersonate legitimate components

### Gorai Deployment Scenarios

| Scenario | Components | Network | Auth Recommendation |
|----------|------------|---------|---------------------|
| **Single-board dev** | All on one Pi | localhost | None (optional token) |
| **Multi-process dev** | Multiple binaries, one machine | localhost | Token or User/Pass |
| **LAN deployment** | Robot + ground station | Private WiFi/Ethernet | NKeys or User/Pass + TLS |
| **Field deployment** | Robot + remote operator | VPN or Internet | NKeys + TLS (required) |
| **Fleet management** | Multiple robots, central server | Internet/VPN | JWT/Accounts + TLS |

---

## Authentication Methods

NATS supports five authentication mechanisms, each with different trade-offs.

### 1. Token Authentication

**Simplest option.** A single shared secret token for all connections.

#### Server Configuration

```hcl
# nats.conf
authorization {
    token: "s3cr3t-t0k3n-h3r3"
}
```

Or via environment variable:
```hcl
authorization {
    token: $NATS_AUTH_TOKEN
}
```

#### Client Connection

```go
// Go client
nc, err := nats.Connect("nats://localhost:4222",
    nats.Token("s3cr3t-t0k3n-h3r3"),
)
```

```bash
# CLI
nats pub test.subject "hello" --server nats://s3cr3t-t0k3n-h3r3@localhost:4222
```

#### Pros/Cons

| Pros | Cons |
|------|------|
| Very simple setup | Single credential for everyone |
| No key management | Token visible in URLs/logs |
| Good for dev/testing | No per-user permissions |
| | Must restart server to change |

#### When to Use

- Local development
- Single-user deployments
- Quick prototyping

---

### 2. User/Password Authentication

**Traditional auth.** Named users with passwords and optional permissions.

#### Server Configuration

```hcl
# nats.conf
authorization {
    users = [
        # Robot main process - full access
        {
            user: robot-main
            password: $ROBOT_MAIN_PASSWORD
            permissions: {
                publish: ">"
                subscribe: ">"
            }
        }

        # Ground station - read-only telemetry, send commands
        {
            user: ground-station
            password: $GROUND_STATION_PASSWORD
            permissions: {
                publish: ["gorai.*.command.>", "gorai.*.request.>"]
                subscribe: ["gorai.>"]
            }
        }

        # Sensor node - publish sensor data only
        {
            user: sensor-node
            password: $SENSOR_NODE_PASSWORD
            permissions: {
                publish: ["gorai.*.sensor.>", "gorai.*.data.>"]
                subscribe: ["gorai.*.config.>"]
            }
        }

        # Read-only monitor
        {
            user: monitor
            password: $MONITOR_PASSWORD
            permissions: {
                subscribe: ["gorai.>"]
                # No publish permissions
            }
        }
    ]
}
```

#### Client Connection

```go
// Go client
nc, err := nats.Connect("nats://localhost:4222",
    nats.UserInfo("robot-main", "password123"),
)
```

```bash
# CLI
nats pub test.subject "hello" --user robot-main --password password123
```

#### Pros/Cons

| Pros | Cons |
|------|------|
| Per-user permissions | Passwords in config files |
| Familiar paradigm | Must restart server to add users |
| Easy to understand | Passwords can be logged/leaked |
| | Centralized user management |

#### When to Use

- Small teams
- LAN deployments with basic security needs
- When you need simple per-user permissions

---

### 3. NKeys Authentication

**Recommended for production.** Public-key cryptography using Ed25519 keys.

NKeys use asymmetric cryptography: the server stores only public keys, while clients hold private keys (seeds). Even if the server is compromised, client credentials remain safe.

#### Key Types

| Prefix | Type | Purpose |
|--------|------|---------|
| `O` | Operator | Signs account JWTs (JWT mode only) |
| `A` | Account | Groups users (JWT mode only) |
| `U` | User | Client identity |
| `N` | Server | Server identity |
| `C` | Cluster | Cluster identity |

For basic NKeys auth, you only need **User** keys (`U` prefix).

#### Generate Keys

```bash
# Install nsc tool
go install github.com/nats-io/nsc/v2@latest

# Generate a user key pair
nsc generate nkey --user

# Output:
# SUAKYRHV...  (seed/private key - keep secret!)
# UABC123...   (public key - put in server config)

# Or use nats CLI
nats nkey gen user
```

#### Server Configuration

```hcl
# nats.conf
authorization {
    users = [
        # Robot main process
        {
            nkey: UABC123ROBOT_PUBLIC_KEY_HERE
            permissions: {
                publish: ">"
                subscribe: ">"
            }
        }

        # Ground station
        {
            nkey: UXYZ789GROUND_STATION_PUBLIC_KEY
            permissions: {
                publish: ["gorai.*.command.>", "gorai.*.request.>"]
                subscribe: ["gorai.>"]
            }
        }

        # Sensor node
        {
            nkey: UDEF456SENSOR_NODE_PUBLIC_KEY
            permissions: {
                publish: ["gorai.*.sensor.>"]
                subscribe: ["gorai.*.config.>"]
            }
        }
    ]
}
```

#### Client Connection

```go
// Go client - using seed directly
nc, err := nats.Connect("nats://localhost:4222",
    nats.Nkey("UABC123...", func(nonce []byte) ([]byte, error) {
        // Sign the nonce with the seed
        kp, _ := nkeys.FromSeed([]byte("SUAKYRHV..."))
        return kp.Sign(nonce)
    }),
)

// Better: load seed from file
seed, _ := os.ReadFile("/etc/gorai/nats-user.nkey")
opt, _ := nats.NkeyOptionFromSeed(string(seed))
nc, err := nats.Connect("nats://localhost:4222", opt)
```

```bash
# CLI
nats pub test.subject "hello" --nkey /path/to/user.nkey
```

#### Seed File Format

The `.nkey` file contains just the seed (private key):

```
SUAKYRHVQNZ3QTYI4VCOIQY5NFEAHCXGXVUFC3EAEVYQHH7YQBPQVLPWUQ
```

#### Pros/Cons

| Pros | Cons |
|------|------|
| Server never sees private keys | Key distribution required |
| Per-user permissions | Slightly more complex setup |
| Keys can be rotated without downtime | Need to manage key files |
| Compromise-resistant | No built-in key revocation |
| Challenge-response (replay resistant) | |

#### When to Use

- Production deployments
- LAN or WAN with security requirements
- When password leakage is a concern
- Multi-device robot systems

---

### 4. JWT/Accounts (Decentralized Authentication)

**Most flexible.** Decentralized identity using signed JWTs. Accounts can be managed independently of the server.

This is the most sophisticated option, suitable for fleet management and multi-tenant scenarios.

#### Concepts

- **Operator**: Root authority, signs account JWTs
- **Account**: Namespace with users and permissions
- **User**: Individual identity within an account

#### Setup with nsc

```bash
# Install nsc
go install github.com/nats-io/nsc/v2@latest

# Create operator
nsc add operator gorai-operator

# Create accounts
nsc add account robot-account
nsc add account ground-station-account
nsc add account monitor-account

# Create users within accounts
nsc add user --account robot-account --name main-process
nsc add user --account ground-station-account --name operator

# Generate credentials file
nsc generate creds --account robot-account --name main-process > robot-main.creds
```

#### Server Configuration

```hcl
# nats.conf - resolver mode
operator: /etc/nats/operator.jwt
resolver: {
    type: full
    dir: /etc/nats/jwt
}
```

Or memory resolver for simpler setups:
```hcl
resolver: MEMORY
resolver_preload: {
    ACCOUNT_PUBLIC_KEY: "eyJ0eXAiOiJqd3Qi..."
}
```

#### Client Connection

```go
// Go client - using credentials file
nc, err := nats.Connect("nats://localhost:4222",
    nats.UserCredentials("/etc/gorai/robot-main.creds"),
)
```

#### Credentials File Format

The `.creds` file contains both JWT and private key:

```
-----BEGIN NATS USER JWT-----
eyJ0eXAiOiJqd3QiLCJhbGciOiJlZDI1NTE5In0...
------END NATS USER JWT------

************************* IMPORTANT *************************
NKEY Seed printed below can be used to sign and prove identity.
NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER NKEY SEED-----
SUAKYRHVQNZ3QTYI4VCOIQY5NFEAHCXGXVUFC3EAEVYQHH7YQBPQVLPWUQ
------END USER NKEY SEED------
```

#### Account Features

JWT accounts provide powerful features:

```bash
# Set account limits
nsc edit account robot-account \
    --conns 100 \           # Max connections
    --data 10GB \           # Max data transfer
    --payload 10MB \        # Max message size
    --exports 50 \          # Max service/stream exports
    --imports 50            # Max imports

# Set user permissions
nsc edit user --account robot-account --name main-process \
    --allow-pub "gorai.>" \
    --allow-sub "gorai.>"

# Export a service for other accounts
nsc add export --account robot-account \
    --name telemetry \
    --subject "gorai.*.telemetry.>" \
    --service

# Import the service in another account
nsc add import --account monitor-account \
    --src-account robot-account \
    --name telemetry \
    --remote-subject "gorai.*.telemetry.>"
```

#### Pros/Cons

| Pros | Cons |
|------|------|
| Decentralized management | Complex initial setup |
| Account isolation | Requires nsc tooling |
| Rich permission model | Overkill for small deployments |
| Can revoke users without server restart | Learning curve |
| Multi-tenant support | |
| Import/export between accounts | |

#### When to Use

- Fleet management (multiple robots)
- Multi-tenant platforms
- When different teams manage different components
- Complex permission requirements
- When you need account-level isolation

---

### 5. TLS Client Certificates (mTLS)

**Certificate-based.** Both server and client authenticate using X.509 certificates.

Can be combined with any of the above methods for transport security + authentication.

#### Certificate Setup

```bash
# Create CA
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 365 -key ca-key.pem -out ca.pem \
    -subj "/CN=Gorai NATS CA"

# Create server certificate
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr \
    -subj "/CN=nats-server"
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem \
    -CAcreateserial -out server-cert.pem -days 365

# Create client certificate (per component)
openssl genrsa -out robot-main-key.pem 4096
openssl req -new -key robot-main-key.pem -out robot-main.csr \
    -subj "/CN=robot-main"
openssl x509 -req -in robot-main.csr -CA ca.pem -CAkey ca-key.pem \
    -CAcreateserial -out robot-main-cert.pem -days 365
```

#### Server Configuration

```hcl
# nats.conf
tls {
    cert_file: "/etc/nats/server-cert.pem"
    key_file: "/etc/nats/server-key.pem"
    ca_file: "/etc/nats/ca.pem"
    verify: true              # Require client certificates
    verify_and_map: true      # Map CN to user for permissions
}

# Optional: map certificate CN to permissions
authorization {
    users = [
        {
            user: "robot-main"    # Matches certificate CN
            permissions: {
                publish: ">"
                subscribe: ">"
            }
        }
    ]
}
```

#### Client Connection

```go
// Go client
nc, err := nats.Connect("nats://localhost:4222",
    nats.ClientCert("/etc/gorai/robot-main-cert.pem", "/etc/gorai/robot-main-key.pem"),
    nats.RootCAs("/etc/gorai/ca.pem"),
)
```

#### Pros/Cons

| Pros | Cons |
|------|------|
| Industry-standard PKI | Certificate management overhead |
| Encrypted transport | Certificate expiration handling |
| Client authentication | Requires CA infrastructure |
| Can map CN to permissions | More complex deployment |
| Hardware security module support | |

#### When to Use

- High-security environments
- Compliance requirements (HIPAA, SOC2)
- When you already have PKI infrastructure
- Combined with other auth methods

---

## Comparison Matrix

| Feature | Token | User/Pass | NKeys | JWT/Accounts | mTLS |
|---------|-------|-----------|-------|--------------|------|
| **Setup Complexity** | Very Low | Low | Medium | High | High |
| **Per-User Permissions** | No | Yes | Yes | Yes | Yes |
| **Secret Exposure Risk** | High | Medium | Low | Low | Low |
| **Key/Credential Rotation** | Restart | Restart | Hot reload | Hot reload | Restart |
| **Offline Provisioning** | No | No | Yes | Yes | Yes |
| **Decentralized Management** | No | No | No | Yes | Partial |
| **Account Isolation** | No | No | No | Yes | No |
| **Transport Encryption** | No | No | No | No | Yes |
| **Replay Protection** | No | No | Yes | Yes | Yes |

---

## Recommendations for Gorai

### Development (Local Machine)

```hcl
# No auth needed - localhost only
port: 4222
host: 127.0.0.1  # Bind to localhost only
```

### Single-Robot LAN Deployment

**Recommended: NKeys + TLS**

```hcl
# nats.conf
port: 4222
host: 0.0.0.0

tls {
    cert_file: "/etc/nats/server-cert.pem"
    key_file: "/etc/nats/server-key.pem"
    ca_file: "/etc/nats/ca.pem"
}

authorization {
    users = [
        {
            nkey: UROBOT_MAIN_PUBLIC_KEY
            permissions: { publish: ">", subscribe: ">" }
        }
        {
            nkey: UGROUND_STATION_PUBLIC_KEY
            permissions: {
                publish: ["gorai.*.command.>", "gorai.*.request.>"]
                subscribe: ["gorai.>"]
            }
        }
    ]
}

jetstream {
    store_dir: "/var/lib/nats/jetstream"
}
```

### Multi-Robot Fleet

**Recommended: JWT/Accounts + TLS**

Each robot gets its own account with isolated namespace:

```
gorai-operator/
├── robot-alpha-account/
│   ├── main-process
│   └── sensor-nodes
├── robot-beta-account/
│   ├── main-process
│   └── sensor-nodes
├── fleet-control-account/
│   └── operator-console
└── monitor-account/
    └── dashboard
```

---

## Implementation Guide

### Step 1: Generate NKeys

```bash
# Create directory for keys
mkdir -p ~/.gorai/nats-keys

# Generate keys for each component
nats nkey gen user > ~/.gorai/nats-keys/robot-main.nkey
nats nkey gen user > ~/.gorai/nats-keys/ground-station.nkey
nats nkey gen user > ~/.gorai/nats-keys/sensor-node.nkey

# Extract public keys for server config
for f in ~/.gorai/nats-keys/*.nkey; do
    echo "$(basename $f .nkey): $(head -1 $f | nats nkey pub)"
done
```

### Step 2: Configure Server

```bash
# Copy public keys to server config
sudo mkdir -p /etc/nats
sudo cp nats/nats.conf /etc/nats/nats.conf

# Edit to add public keys
sudo vim /etc/nats/nats.conf
```

### Step 3: Update Gorai Client Code

The `pkg/nats/client.go` needs to be updated to support authentication:

```go
// Config holds NATS connection configuration.
type Config struct {
    URL             string
    Name            string
    ConnectTimeout  time.Duration
    ReconnectWait   time.Duration
    MaxReconnects   int

    // Authentication
    Token           string  // Token auth
    Username        string  // User/pass auth
    Password        string
    NKeyFile        string  // NKey seed file path
    CredentialsFile string  // JWT credentials file path

    // TLS
    TLSCert   string  // Client certificate path
    TLSKey    string  // Client key path
    TLSCA     string  // CA certificate path
    TLSVerify bool    // Verify server certificate
}
```

### Step 4: Distribute Credentials

```bash
# On robot
scp ~/.gorai/nats-keys/robot-main.nkey robot:/etc/gorai/
chmod 600 /etc/gorai/robot-main.nkey

# On ground station
scp ~/.gorai/nats-keys/ground-station.nkey laptop:/etc/gorai/
chmod 600 /etc/gorai/ground-station.nkey
```

---

## RDL Configuration

### Current Schema (Extended)

```json
{
  "version": "3",
  "robot": {
    "name": "my-robot"
  },
  "nats": {
    "url": "nats://nats-server:4222",
    "jetstream": true,

    "auth": {
      "method": "nkey",
      "nkey_file": "/etc/gorai/robot-main.nkey"
    },

    "tls": {
      "enabled": true,
      "ca_file": "/etc/gorai/ca.pem",
      "cert_file": "/etc/gorai/robot-cert.pem",
      "key_file": "/etc/gorai/robot-key.pem",
      "verify": true
    }
  }
}
```

### Auth Method Options

```json
// Token
"auth": {
  "method": "token",
  "token": "$NATS_AUTH_TOKEN"
}

// User/Password
"auth": {
  "method": "userpass",
  "username": "robot-main",
  "password": "$NATS_PASSWORD"
}

// NKey (recommended)
"auth": {
  "method": "nkey",
  "nkey_file": "/etc/gorai/robot.nkey"
}

// JWT Credentials
"auth": {
  "method": "credentials",
  "credentials_file": "/etc/gorai/robot.creds"
}
```

---

## Server Configuration

### Production Template

```hcl
# /etc/nats/nats.conf - Production Gorai NATS Server

server_name: gorai-nats
port: 4222
host: 0.0.0.0

# Monitoring (bind to localhost for security)
http_port: 8222
http: 127.0.0.1

# Limits
max_payload: 10485760  # 10MB for images
max_connections: 256
max_subscriptions: 0   # unlimited

# Logging
logtime: true
log_file: "/var/log/nats/nats.log"

# TLS (required for production)
tls {
    cert_file: "/etc/nats/server-cert.pem"
    key_file: "/etc/nats/server-key.pem"
    ca_file: "/etc/nats/ca.pem"
    timeout: 2.0
}

# NKey Authentication
authorization {
    users = [
        # Robot main process - full access
        {
            nkey: UABC123_ROBOT_MAIN_PUBLIC_KEY
            permissions: {
                publish: ">"
                subscribe: ">"
            }
        }

        # Ground station - command + telemetry
        {
            nkey: UXYZ789_GROUND_STATION_PUBLIC_KEY
            permissions: {
                publish: {
                    allow: ["gorai.*.command.>", "gorai.*.request.>", "_INBOX.>"]
                }
                subscribe: {
                    allow: ["gorai.>", "_INBOX.>"]
                }
            }
        }

        # Read-only monitor
        {
            nkey: UDEF456_MONITOR_PUBLIC_KEY
            permissions: {
                subscribe: {
                    allow: ["gorai.>"]
                }
            }
        }
    ]
}

# JetStream
jetstream {
    store_dir: "/var/lib/nats/jetstream"
    max_memory_store: 1GB
    max_file_store: 50GB
}
```

### Permission Patterns for Gorai

```hcl
# Full robot control
permissions: {
    publish: ">"
    subscribe: ">"
}

# Ground station (can send commands, receive everything)
permissions: {
    publish: {
        allow: [
            "gorai.*.command.>",      # Motor, servo commands
            "gorai.*.request.>",      # RPC requests
            "gorai.mesh.>",           # Mesh queries
            "_INBOX.>"                # Request-reply inbox
        ]
    }
    subscribe: {
        allow: ["gorai.>", "_INBOX.>"]
    }
}

# Sensor node (can only publish sensor data)
permissions: {
    publish: {
        allow: [
            "gorai.*.sensor.>",
            "gorai.*.data.>",
            "gorai.mesh.announce"
        ]
    }
    subscribe: {
        allow: [
            "gorai.*.config.>",
            "gorai.mesh.heartbeat.*"
        ]
    }
}

# Monitor (read-only, no JetStream write)
permissions: {
    subscribe: {
        allow: ["gorai.>"]
        deny: ["$JS.>"]  # No JetStream admin
    }
}
```

---

## Credential Management

### Key Storage Best Practices

| Location | Use Case | Protection |
|----------|----------|------------|
| `/etc/gorai/*.nkey` | Production robots | `chmod 600`, owned by service user |
| `~/.gorai/nats-keys/` | Development | `chmod 600` |
| Environment variable | CI/CD, containers | Secret management system |
| Hardware security module | High-security | HSM/TPM integration |

### Key Rotation

```bash
# 1. Generate new key
nats nkey gen user > new-robot.nkey

# 2. Add new key to server config (keep old key)
authorization {
    users = [
        { nkey: OLD_KEY, permissions: {...} }
        { nkey: NEW_KEY, permissions: {...} }
    ]
}

# 3. Reload server config (no restart needed)
nats-server --signal reload

# 4. Update robot with new key and restart

# 5. Remove old key from server config
# 6. Reload server again
```

### Environment Variables

For containerized deployments, pass credentials via environment:

```bash
# Set in environment
export NATS_NKEY_SEED="SUAKYRHV..."

# In RDL
"auth": {
  "method": "nkey",
  "nkey_seed": "$NATS_NKEY_SEED"
}
```

---

## Security Best Practices

### Network Security

1. **Use TLS** for any non-localhost connections
2. **Bind to specific interfaces** - don't use `0.0.0.0` without auth
3. **Firewall** - only allow NATS port from known IPs
4. **VPN** - for remote connections, use WireGuard or similar

### Credential Security

1. **Never commit credentials** to git
2. **Use environment variables** or secret managers in production
3. **Rotate keys** periodically (quarterly for production)
4. **Audit access** - monitor who connects

### Permission Minimization

1. **Principle of least privilege** - only grant needed permissions
2. **Separate accounts** for different trust levels
3. **Deny lists** for sensitive subjects
4. **No wildcards** for untrusted components

### Monitoring

```bash
# Monitor connections
nats server report connections

# Check for unauthorized attempts (in logs)
grep "Authentication" /var/log/nats/nats.log

# Monitor with Prometheus
# NATS exposes metrics at /metrics
```

---

## Troubleshooting

### "Authorization Violation"

```
Error: nats: Authorization Violation
```

**Causes:**
- Wrong token/password/key
- Missing permissions for subject
- Key not in server config

**Debug:**
```bash
# Check if key is valid
nats nkey pub < /path/to/user.nkey

# Test connection
nats pub test.subject "hello" --nkey /path/to/user.nkey -v
```

### "TLS Handshake Failed"

```
Error: tls: failed to verify certificate
```

**Causes:**
- CA not trusted
- Certificate expired
- Hostname mismatch

**Debug:**
```bash
# Check certificate
openssl x509 -in server-cert.pem -text -noout

# Test TLS connection
openssl s_client -connect nats-server:4222 -CAfile ca.pem
```

### "No Responders Available"

With authentication, request-reply requires `_INBOX.>` permissions:

```hcl
permissions: {
    publish: {
        allow: ["gorai.*.request.>", "_INBOX.>"]
    }
    subscribe: {
        allow: ["gorai.>", "_INBOX.>"]
    }
}
```

### Key Format Issues

NKey seeds must be exact - no extra whitespace:

```bash
# Check for whitespace
cat -A /path/to/user.nkey

# Should show just the key, no trailing ^M or spaces
SUAKYRHVQNZ3QTYI4VCOIQY5NFEAHCXGXVUFC3EAEVYQHH7YQBPQVLPWUQ$
```

---

## Next Steps

1. **Choose auth method** based on deployment scenario
2. **Generate credentials** for each component
3. **Update server config** with public keys/users
4. **Update RDL configs** with auth settings
5. **Test connectivity** before deploying
6. **Enable TLS** for production

---

## References

- [NATS Security Documentation](https://docs.nats.io/running-a-nats-service/configuration/securing_nats)
- [NKeys Specification](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/nkey_auth)
- [JWT/Accounts Guide](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/jwt)
- [nsc Tool Documentation](https://docs.nats.io/using-nats/nats-tools/nsc)
- [NATS TLS Configuration](https://docs.nats.io/running-a-nats-service/configuration/securing_nats/tls)
