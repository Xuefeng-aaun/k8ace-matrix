# Local Registry

This optional registry is intended for weak-network environments where pushing
directly to Docker Hub or another remote registry is unstable.

Start it:

```bash
docker compose -f hack/local-registry/docker-compose.yaml up -d
```

Use it from a local override:

```yaml
registry_prefix: "registry.local:5000/k8ace"

ci_cd:
  argo_workflows:
    insecure_registries:
      - "registry.local:5000"
```

For Kubernetes builders, make sure cluster nodes can resolve and reach the
registry address. If the registry is plain HTTP, configure the node runtime or
Kaniko `insecure_registries` accordingly before production use.
