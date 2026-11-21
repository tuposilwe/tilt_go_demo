# Build the Docker image
docker_build(
    'tilt-go-backend',
    './backend'
)

# Load Kubernetes configurations
k8s_yaml([
    'k8s/deployment.yaml',
    'k8s/service.yaml'
])

# Configure the backend resource
k8s_resource(
    'go-backend',
    port_forwards='8080:8080'
)