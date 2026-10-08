# Operations Guide

## Deployment

### Docker Compose (Development/Staging)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

### Docker (Production)

```bash
# Build image
docker build -t forgestock:latest .

# Run API
docker run -d \
  --name forgestock-api \
  -p 8080:8080 \
  -e DATABASE_URL=postgres://user:pass@host:5432/db \
  -e REDIS_URL=redis://host:6379/0 \
  -e JWT_SECRET=your-secret \
  forgestock:latest

# Run Worker
docker run -d \
  --name forgestock-worker \
  -e DATABASE_URL=postgres://user:pass@host:5432/db \
  -e REDIS_URL=redis://host:6379/0 \
  -e JWT_SECRET=your-secret \
  forgestock:latest ./forgestock-worker
```

### Kubernetes

See `deployments/k8s/` for manifests.

```bash
# Apply manifests
kubectl apply -f deployments/k8s/

# Check status
kubectl get pods -l app=forgestock

# View logs
kubectl logs -l app=forgestock-api
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| APP_ENV | No | Environment (development/production) |
| HTTP_PORT | No | HTTP port (default: 8080) |
| DATABASE_URL | Yes | PostgreSQL connection string |
| REDIS_URL | Yes | Redis connection string |
| JWT_SECRET | Yes | JWT signing secret (min 32 chars) |
| ACCESS_TOKEN_TTL | No | Access token TTL (default: 15m) |
| REFRESH_TOKEN_TTL | No | Refresh token TTL (default: 168h) |
| LOG_LEVEL | No | Log level (debug/info/warn/error) |
| RATE_LIMIT_ENABLED | No | Enable rate limiting (default: true) |
| RATE_LIMIT_RPS | No | Requests per second (default: 100) |
| CORS_ALLOWED_ORIGINS | No | Comma-separated origins (default: *) |

### Secrets Management

**Never commit secrets to git.**

Options:
1. **Docker secrets**: `docker secret create jwt_secret secret.txt`
2. **Kubernetes secrets**: `kubectl create secret generic forgestock --from-literal=JWT_SECRET=...`
3. **External secret manager**: HashiCorp Vault, AWS Secrets Manager, etc.
4. **Environment files**: `.env` (gitignored)

## Database Operations

### Backup

```bash
# Logical backup
pg_dump -h localhost -U forgestock -d forgestock > backup.sql

# Compressed
pg_dump -h localhost -U forgestock -d forgestock | gzip > backup.sql.gz

# Custom format (parallel restore)
pg_dump -Fc -h localhost -U forgestock -d forgestock > backup.dump
```

### Restore

```bash
# From SQL
psql -h localhost -U forgestock -d forgestock < backup.sql

# From custom format
pg_restore -h localhost -U forgestock -d forgestock backup.dump
```

### Point-in-Time Recovery

Configure WAL archiving in `postgresql.conf`:
```
wal_level = replica
archive_mode = on
archive_command = 'cp %p /path/to/archive/%f'
```

### Maintenance

```bash
# Vacuum analyze
psql -h localhost -U forgestock -d forgestock -c "VACUUM ANALYZE;"

# Reindex (if needed)
psql -h localhost -U forgestock -d forgestock -c "REINDEX DATABASE forgestock;"

# Check table sizes
psql -h localhost -U forgestock -d forgestock -c "
  SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
  FROM pg_tables WHERE schemaname = 'public'
  ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
"
```

## Monitoring

### Health Checks

```bash
# Liveness
curl http://localhost:8080/api/v1/health

# Readiness
curl http://localhost:8080/api/v1/ready
```

Expected responses:
- `/health`: 200 OK with `{"status": "healthy"}`
- `/ready`: 200 OK with `{"status": "ready"}` or 503 with `{"status": "ready", "checks": {"database": "not ready"}}`

### Metrics

Prometheus metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

Key metrics to alert on:
- `http_requests_total{status=~"5.."}` - 5xx errors
- `http_request_duration_seconds` - p99 latency
- `database_operations_total{status="error"}` - DB errors
- `background_jobs_total{status="failed"}` - Failed jobs

### Logging

Structured JSON logs to stdout. Configure log aggregation:

**Fluentd/Fluent Bit:**
```xml
<source>
  @type forward
  port 24224
</source>

<match forgestock.**>
  @type elasticsearch
  host elasticsearch
  port 9200
  index_name forgestock-logs
</match>
```

**Log fields:**
- `timestamp` - RFC3339
- `level` - debug/info/warn/error
- `message` - Human readable
- `request_id` - Correlation ID
- `audit_*` - Audit log fields

### Alerting Rules (Prometheus)

```yaml
groups:
- name: forgestock
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "High 5xx error rate"
      
  - alert: HighLatency
    expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High p99 latency"
      
  - alert: DatabaseDown
    expr: up{job="forgestock"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "ForgeStock API down"
      
  - alert: WorkerQueueBacklog
    expr: redis_list_length{name="notifications"} > 1000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Notification queue backlog"
```

## Scaling

### Horizontal Scaling

API servers are stateless - add more replicas behind load balancer.

```yaml
# Kubernetes HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: forgestock-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: forgestock-api
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Database Scaling

1. **Read replicas** for read-heavy workloads
2. **Connection pooling** (PgBouncer) for many clients
3. **Partitioning** for large tables (audit_logs by date)
4. **Caching** frequently accessed data in Redis

### Redis Scaling

1. **Cluster mode** for high availability
2. **Separate databases** for different purposes (queue, cache, sessions)
3. **Memory policies**: `allkeys-lru` for cache, `noeviction` for queue

## Troubleshooting

### API Not Responding

```bash
# Check process
ps aux | grep forgestock

# Check logs
docker logs forgestock-api

# Check health
curl -v http://localhost:8080/api/v1/health

# Check ports
netstat -tlnp | grep 8080
```

### Database Connection Issues

```bash
# Test connection
psql $DATABASE_URL -c "SELECT 1;"

# Check max connections
psql $DATABASE_URL -c "SHOW max_connections;"

# Check active connections
psql $DATABASE_URL -c "SELECT count(*) FROM pg_stat_activity;"
```

### Redis Connection Issues

```bash
# Test connection
redis-cli -u $REDIS_URL ping

# Check memory
redis-cli -u $REDIS_URL info memory

# Check connected clients
redis-cli -u $REDIS_URL client list
```

### High Memory Usage

```bash
# Check Go memory stats
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check for goroutine leaks
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof
```

### Slow Queries

```bash
# Enable pg_stat_statements
# In postgresql.conf:
# shared_preload_libraries = 'pg_stat_statements'
# pg_stat_statements.track = all

# Query slow queries
psql $DATABASE_URL -c "
  SELECT query, calls, total_time, mean_time, rows
  FROM pg_stat_statements
  ORDER BY mean_time DESC
  LIMIT 20;
"
```

## Security Operations

### Rotate JWT Secret

1. Generate new secret: `openssl rand -base64 32`
2. Update `JWT_SECRET` in all environments
3. Deploy new version
4. All existing tokens become invalid (users must re-login)
5. Consider gradual rollout with two secrets during transition

### Certificate Rotation

```bash
# Generate new TLS cert
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Update load balancer / ingress
kubectl create secret tls forgestock-tls --cert=cert.pem --key=key.pem
```

### Audit Log Review

```bash
# Recent admin actions
psql $DATABASE_URL -c "
  SELECT created_at, actor_id, action, entity_type, entity_id
  FROM audit_logs
  WHERE actor_type = 'admin'
  ORDER BY created_at DESC
  LIMIT 50;
"

# Failed logins
psql $DATABASE_URL -c "
  SELECT created_at, metadata->>'email' as email, metadata->>'reason' as reason
  FROM audit_logs
  WHERE action = 'login_failed'
  ORDER BY created_at DESC
  LIMIT 50;
"
```

## Incident Response

### Runbook: API Down

1. Check health endpoint
2. Check container/pod status
3. Check logs for errors
4. Check database connectivity
5. Check Redis connectivity
6. Restart if needed: `docker restart forgestock-api`
7. If persistent, check recent deployments

### Runbook: Database Slow

1. Check active queries: `SELECT * FROM pg_stat_activity WHERE state = 'active';`
2. Check for blocking: `SELECT * FROM pg_locks WHERE NOT granted;`
3. Check slow queries (pg_stat_statements)
4. Consider killing long-running queries
5. Scale read replicas if read-heavy

### Runbook: Queue Backlog

1. Check worker status: `docker ps | grep worker`
2. Check worker logs for errors
3. Check queue depth: `redis-cli LLEN notifications`
4. Restart worker if stuck
5. Consider adding more worker replicas

## Maintenance Windows

### Scheduled Maintenance

1. Announce 48 hours in advance
2. Disable new deployments
3. Run during low-traffic period
4. Steps:
   - Backup database
   - Run migrations
   - Deploy new version
   - Run smoke tests
   - Re-enable deployments

### Zero-Downtime Deployments

1. Blue-green or rolling deployment
2. Health checks must pass before traffic routed
3. Database migrations must be backward compatible
4. Run migrations before deploying new code
5. Use `pg_dump --no-owner --no-acl` for schema changes

## Disaster Recovery

### RPO/RTO Targets

| Tier | RPO | RTO |
|------|-----|-----|
| Critical (orders, payments) | < 1 min | < 15 min |
| Important (inventory, users) | < 1 hour | < 1 hour |
| Standard (audit, notifications) | < 24 hours | < 4 hours |

### Recovery Procedures

1. **Database restore**: Restore from latest backup + WAL replay
2. **Redis restore**: Restore from RDB/AOF backup
3. **Application redeploy**: Deploy from known-good image
4. **DNS failover**: Update DNS to backup region

### Testing DR

- Quarterly restore tests
- Document recovery time
- Update runbooks based on results