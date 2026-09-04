# Deployment recipes

These examples run hello as a small HTTP backend. Pin a complete version or an
image digest in repeatable environments; `latest` is intended for interactive
evaluation, not deployment automation.

## Image tags

Version releases publish the same linux/amd64 and linux/arm64 image to Docker
Hub and GitHub Container Registry:

| Tag | Meaning |
| --- | --- |
| `2.3.0` | Immutable release version; preferred for deployments. |
| `2.2` / `2` | Moving minor and major aliases. |
| `latest` | Most recently published release. |
| `main` | Latest successful build from the default branch. |

Use `ghcr.io/soulteary/hello:2.3.0` or `soulteary/hello:2.3.0`. See
[`verification.md`](verification.md) before promoting a new version.

## Docker Compose

```yaml
services:
  hello:
    image: ghcr.io/soulteary/hello:2.3.0
    command: ["-listen", ":8080", "-http-max-streams", "64"]
    ports:
      - "127.0.0.1:8080:8080"
    read_only: true
    cap_drop: ["ALL"]
    security_opt:
      - no-new-privileges:true
    restart: unless-stopped
```

The image is distroless and contains no shell, curl or wget. Check
`http://127.0.0.1:8080/healthz` from the host or orchestrator rather than using
an exec-based container health check.

## Nginx subpath

This exposes the service at `https://example.com/hello/`. The trailing slash on
`proxy_pass` strips the public prefix before forwarding to hello.

```nginx
location = /hello {
    return 308 /hello/;
}

location /hello/ {
    proxy_pass http://hello:8080/;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Host $host;
    proxy_set_header X-Forwarded-Prefix /hello;
    proxy_set_header X-Forwarded-Proto $scheme;

    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 1h;
}
```

The browser derives `/hello/events` from its public URL, while Nginx forwards
that request to the backend `/events` endpoint. Disable compression or other
response aggregation for this route if an outer proxy still buffers events.

## Traefik subpath

```yaml
services:
  hello:
    image: ghcr.io/soulteary/hello:2.3.0
    command: ["-listen", ":8080"]
    labels:
      - traefik.enable=true
      - traefik.http.routers.hello.rule=Host(`example.com`) && PathPrefix(`/hello`)
      - traefik.http.routers.hello.middlewares=hello-strip
      - traefik.http.middlewares.hello-strip.stripprefix.prefixes=/hello
      - traefik.http.services.hello.loadbalancer.server.port=8080
```

Configure the forwarding timeout on the Traefik entry point to exceed the
expected browser session. hello emits `X-Accel-Buffering: no` and periodic SSE
comments, but proxy timeouts remain authoritative.

## Kubernetes probes and resources

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
spec:
  replicas: 1
  selector:
    matchLabels: {app: hello}
  template:
    metadata:
      labels: {app: hello}
    spec:
      containers:
        - name: hello
          image: ghcr.io/soulteary/hello:2.3.0
          args: ["-listen", ":8080", "-http-max-streams", "64"]
          ports:
            - {name: http, containerPort: 8080}
          readinessProbe:
            httpGet: {path: /healthz, port: http}
          livenessProbe:
            httpGet: {path: /healthz, port: http}
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            capabilities: {drop: ["ALL"]}
          resources:
            requests: {cpu: 10m, memory: 8Mi}
            limits: {cpu: 250m, memory: 64Mi}
---
apiVersion: v1
kind: Service
metadata:
  name: hello
spec:
  selector: {app: hello}
  ports:
    - {name: http, port: 8080, targetPort: http}
```

Tune `-http-max-streams` and resource limits together. A full stream slot
returns `503 Service Unavailable` with `Retry-After: 1`; health checks remain
available and do not consume a stream slot.

## Diagnostic privacy

By default, text responses omit the URL query string and identity-bearing
headers. `-reflect-query` and `-reflect-identity` are intended only for a
controlled debugging environment. A reverse proxy must remove client-supplied
`X-Auth-*`, `X-Forwarded-User` and related identity headers before setting its
own trusted values.

The hostname remains enabled in the command-line server by default because it
is useful for load-balancing demos. Disable it with `-reflect-hostname=false`
when infrastructure identifiers should not appear in responses.
