
# # Restart Go process after Live Update
# load('ext://restart_process', 'docker_build_with_restart')

# # ===================================
# # 1. BUILD GO BACKEND
# # ===================================

# docker_build_with_restart(
#     'tilt-go-backend',
#     './backend',

#     entrypoint='/app/server',

#     live_update=[
#         sync('./backend/main.go', '/app/main.go'),

#         run('cd /app && go build -o /app/server .'),
#     ],
# )

# # ===================================
# # 2. DEPLOY POSTGRES HELM CHART
# # ===================================

# postgres_yaml = helm(
#     './charts/postgres',
#     name='postgres',
#     values=[
#         './charts/postgres/values.yaml',
#     ],
# )

# k8s_yaml(postgres_yaml)

# # ===================================
# # 3. DEPLOY BACKEND HELM CHART
# # ===================================

# backend_yaml = helm(
#     './charts/backend',
#     name='go-backend',
#     values=[
#         './charts/backend/values.yaml',
#     ],
# )

# k8s_yaml(backend_yaml)

# # ===================================
# # 4. CONFIGURE RESOURCES
# # ===================================

# k8s_resource(
#     'postgres',
#     port_forwards='5432:5432',
# )

# k8s_resource(
#     'go-backend',
#     port_forwards='8080:8080',
#     resource_deps=['postgres'],
# )



load('ext://restart_process', 'docker_build_with_restart')

# ===================================
# 1. BUILD GO BACKEND
# ===================================

docker_build_with_restart(
    'tilt-go-backend',
    './backend',
    entrypoint='/app/server',
    live_update=[
        sync('./backend/main.go', '/app/main.go'),
        run('cd /app && go build -o /app/server .'),
    ],
)

# ===================================
# 2. DEPLOY POSTGRES HELM CHART
# ===================================

# Tilt 0.33's helm() checks `helm version --client`, which Helm 4 no longer supports.
watch_file('./charts/postgres')
k8s_yaml(local('helm template postgres ./charts/postgres'))

# ===================================
# 3. DEPLOY BACKEND HELM CHART
# ===================================

watch_file('./charts/backend')
k8s_yaml(local('helm template go-backend ./charts/backend'))

# ===================================
# 4. CONFIGURE RESOURCES
# ===================================

k8s_resource(
    'postgres',
    port_forwards='5432:5432',
)

k8s_resource(
    'go-backend',
    port_forwards='8080:8080',
    resource_deps=['postgres'],
)
