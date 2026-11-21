
# Import Tilt's process restart extension
load('ext://restart_process', 'docker_build_with_restart')

# Build backend with Live Update
docker_build_with_restart(
    'tilt-go-backend',
    './backend',

    # Command to start and restart after Live Update
    entrypoint='/app/server',

    live_update=[
        # 1. Sync changed Go source file
        sync('./backend/main.go', '/app/main.go'),

        # 2. Compile updated Go application
        run('cd /app && go build -o /app/server .'),
    ],
)

# Deploy PostgreSQL first
k8s_yaml('database/postgres.yaml')

# Deploy backend
k8s_yaml([
    'k8s/deployment.yaml',
    'k8s/service.yaml',
])

# PostgreSQL resource
k8s_resource(
    'postgres',
    port_forwards='5432:5432',
)

# Backend depends on PostgreSQL
k8s_resource(
    'go-backend',
    port_forwards='8080:8080',
    resource_deps=['postgres'],
)
