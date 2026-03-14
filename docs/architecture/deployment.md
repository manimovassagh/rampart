# Deployment Architecture

Rampart is designed as a single binary that runs anywhere — from a single EC2 instance to a Kubernetes cluster. Cloud-agnostic by design, it deploys equally well on AWS, Azure, GCP, or on-premises.

## Deployment Diagram

```mermaid
graph TB
    subgraph "Cloud Provider (AWS / Azure / GCP / On-Prem)"
        subgraph "Load Balancer"
            LB[ALB / Azure LB / GCP LB / Nginx]
        end

        subgraph "Compute"
            R1[Rampart Instance 1]
            R2[Rampart Instance 2]
            R3[Rampart Instance N]
        end

        subgraph "Data Layer"
            PG[(PostgreSQL)]
        end

        subgraph "External Services"
            SMTP[Email Service]
            SMS[SMS Service]
        end
    end

    Internet((Internet)) --> LB
    LB --> R1
    LB --> R2
    LB --> R3

    R1 --> PG
    R2 --> PG
    R3 --> PG

    R1 --> SMTP
    R1 --> SMS
```

## Deployment Options

### Option 1: Single Instance (Simplest)

A single Rampart binary on a VM. Suitable for small teams, dev/staging environments, or low-traffic production setups.

```
EC2 / Azure VM / GCE instance
├── rampart binary (port 8080)
└── PostgreSQL (same host or managed)
```

**When to use**: < 1000 users, single region, simplicity is priority.

### Option 2: Container (Docker / ECS / Cloud Run / Azure Container Instances)

Run Rampart as a container with managed database and cache.

```yaml
# docker-compose.yml (production-ready)
services:
  rampart:
    image: ghcr.io/manimovassagh/rampart:latest
    ports:
      - "8080:8080"
    environment:
      RAMPART_DB_URL: postgres://user:pass@db:5432/rampart?sslmode=require
      RAMPART_ISSUER: https://auth.example.com
      RAMPART_LOG_LEVEL: info
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 128M
          cpus: "0.5"

  db:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data

```

### Option 3: Kubernetes

Helm chart (provided) for Kubernetes deployments with horizontal scaling.

```yaml
# values.yaml highlights
replicaCount: 3
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "128Mi"
    cpu: "500m"

postgresql:
  enabled: true  # or use external

ingress:
  enabled: true
  hosts:
    - host: auth.example.com
      paths: ["/"]
  tls:
    - secretName: auth-tls
      hosts: [auth.example.com]
```

## Cloud-Specific Deployment Guides

### AWS

Rampart works with any AWS compute option:

| Service | Best For |
|---------|----------|
| **EC2** | Single instance, full control, lowest cost |
| **ECS Fargate** | Containerized, no server management, auto-scaling |
| **EKS** | Kubernetes, large-scale, multi-service |
| **App Runner** | Simplest container deployment |

**Recommended infrastructure:**
- **Compute**: ECS Fargate or EC2
- **Database**: Amazon RDS for PostgreSQL
- **Load Balancer**: Application Load Balancer (ALB)
- **DNS**: Route 53
- **TLS**: AWS Certificate Manager
- **Secrets**: AWS Secrets Manager

### Azure

| Service | Best For |
|---------|----------|
| **Azure VM** | Single instance, full control |
| **Azure Container Instances** | Simple container deployment |
| **Azure Container Apps** | Serverless containers, auto-scaling |
| **AKS** | Kubernetes, large-scale |

**Recommended infrastructure:**
- **Compute**: Azure Container Apps or VM
- **Database**: Azure Database for PostgreSQL
- **Load Balancer**: Azure Application Gateway
- **DNS**: Azure DNS
- **TLS**: Azure Key Vault
- **Secrets**: Azure Key Vault

### Google Cloud

| Service | Best For |
|---------|----------|
| **Compute Engine** | Single instance, full control |
| **Cloud Run** | Serverless containers, auto-scaling, simplest |
| **GKE** | Kubernetes, large-scale |

**Recommended infrastructure:**
- **Compute**: Cloud Run or Compute Engine
- **Database**: Cloud SQL for PostgreSQL
- **Load Balancer**: Cloud Load Balancing
- **DNS**: Cloud DNS
- **TLS**: Google-managed certificates
- **Secrets**: Secret Manager

## Terraform Examples

Rampart can be deployed with Terraform on any cloud provider. Reference modules will be provided in `deploy/terraform/`.

### AWS — ECS Fargate

```hcl
module "rampart" {
  source = "github.com/manimovassagh/rampart//deploy/terraform/aws"

  name        = "rampart"
  environment = "production"

  # Compute
  cpu    = 512
  memory = 1024
  desired_count = 2

  # Networking
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnet_ids

  # Database
  db_instance_class = "db.t4g.medium"
  db_name           = "rampart"

  # DNS
  domain_name = "auth.example.com"
  zone_id     = data.aws_route53_zone.main.zone_id

  # Configuration
  rampart_config = {
    RAMPART_ISSUER    = "https://auth.example.com"
    RAMPART_LOG_LEVEL = "info"
  }
}
```

### AWS — Simple EC2

```hcl
module "rampart_ec2" {
  source = "github.com/manimovassagh/rampart//deploy/terraform/aws-ec2"

  name          = "rampart"
  instance_type = "t4g.small"
  ami_id        = data.aws_ami.ubuntu.id
  key_name      = "my-key"

  vpc_id    = module.vpc.vpc_id
  subnet_id = module.vpc.public_subnet_ids[0]

  domain_name = "auth.example.com"

  # Rampart will be installed via user_data script
  # with PostgreSQL on the same instance
}
```

### Azure — Container Apps

```hcl
module "rampart" {
  source = "github.com/manimovassagh/rampart//deploy/terraform/azure"

  name                = "rampart"
  resource_group_name = azurerm_resource_group.main.name
  location            = "eastus"

  # Container
  container_image = "ghcr.io/manimovassagh/rampart:latest"
  cpu             = 0.5
  memory          = "1Gi"
  min_replicas    = 1
  max_replicas    = 5

  # Database
  postgresql_sku = "B_Standard_B1ms"

  # DNS
  custom_domain = "auth.example.com"
}
```

## Configuration

Rampart is configured via environment variables or a YAML config file. Environment variables take precedence.

### Required Configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `RAMPART_DB_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/rampart?sslmode=require` |
| `RAMPART_ISSUER` | The issuer URL (your public domain) | `https://auth.example.com` |

### Optional Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `RAMPART_PORT` | `8080` | HTTP listen port |
| `RAMPART_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `RAMPART_LOG_FORMAT` | `json` | Log format (json, text) |
| `RAMPART_TLS_CERT` | — | Path to TLS certificate (if terminating TLS at app) |
| `RAMPART_TLS_KEY` | — | Path to TLS private key |
| `RAMPART_MASTER_KEY` | auto-generated | AES-256 key for encrypting secrets at rest |
| `RAMPART_SMTP_URL` | — | SMTP connection string for email |
| `RAMPART_METRICS_ENABLED` | `false` | Enable Prometheus `/metrics` endpoint |

### YAML Config

```yaml
# config.yaml
server:
  port: 8080
  issuer: https://auth.example.com

database:
  url: postgres://user:pass@host:5432/rampart?sslmode=require
  max_open_conns: 25
  max_idle_conns: 5

logging:
  level: info
  format: json

security:
  master_key: "${RAMPART_MASTER_KEY}"  # reference env var
  bcrypt_cost: 12
  argon2_memory: 65536
  argon2_iterations: 3
  argon2_parallelism: 2
```

## Health Checks

All deployment platforms should use these endpoints for health monitoring:

| Endpoint | Purpose | Use |
|----------|---------|-----|
| `GET /healthz` | Process is alive | Liveness probe |
| `GET /readyz` | DB connected, ready for traffic | Readiness probe |

## Resource Requirements

| Scenario | CPU | Memory | Disk |
|----------|-----|--------|------|
| Dev / small team | 0.25 vCPU | 64 MB | 1 GB |
| Production (< 10k users) | 0.5 vCPU | 128 MB | 10 GB |
| Production (< 100k users) | 1 vCPU | 256 MB | 50 GB |
| Enterprise (100k+ users) | 2+ vCPU (scaled horizontally) | 512 MB per instance | 100+ GB |

Rampart is stateless (all state in PostgreSQL), so horizontal scaling is straightforward — add more instances behind the load balancer.
