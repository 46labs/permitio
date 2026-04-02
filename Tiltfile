load('ext://helm_resource', 'helm_resource')
load('ext://namespace', 'namespace_create')

allow_k8s_contexts('kind-permitio')

namespace_create('permitio')

update_settings(k8s_upsert_timeout_secs=120)

# Build permitio image
docker_build(
    'ghcr.io/46labs/permitio',
    '.',
    dockerfile='./Dockerfile',
    live_update=[
        sync('./pkg', '/app/pkg'),
        sync('./cmd', '/app/cmd'),
        run('go build -o /app/permitio cmd/main.go', trigger=['./go.mod', './go.sum']),
    ],
)

# Deploy with Helm
k8s_yaml(helm(
    './charts/permitio',
    name='permitio',
    namespace='permitio',
    set=[
        'service.type=ClusterIP',
        'ingress.enabled=false',
    ],
))

k8s_resource(
    'permitio',
    labels=['permitio'],
)

# Health check
local_resource(
    'health-check',
    cmd='curl -sf http://localhost:7766/v2/schema/resources | jq length || echo "Permitio not ready"',
    auto_init=False,
    labels=['helpers'],
)
