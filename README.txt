TILT GO DEMO
============

This project runs a Go HTTP backend and PostgreSQL on Kubernetes. Tilt builds
the backend image, deploys the PostgreSQL and backend Helm charts, and
automatically updates the backend when backend/main.go changes.

The local development setup uses:
  - Docker
  - kubectl
  - kind
  - Helm
  - Tilt 0.33.21


INSTALL TOOLS (LINUX)
=====================

Install kubectl:

  sudo snap install kubectl --classic

Install Tilt 0.33.21:

  PROTO="https://"
  HOST="github.com"
  PATH_URL="/tilt-dev/tilt/releases/download/v0.33.21/tilt.0.33.21.linux.x86_64.tar.gz"

  curl -fsSL -o tilt.tar.gz "${PROTO}${HOST}${PATH_URL}"
  tar -xzf tilt.tar.gz tilt
  sudo mv tilt /usr/local/bin/tilt
  rm -f tilt.tar.gz

Install or reinstall kind v0.25.0:

  PROTO="https://"
  HOST="kind.sigs.k8s.io"
  PATH_URL="/dl/v0.25.0/kind-linux-amd64"

  curl -Lo ./kind "${PROTO}${HOST}${PATH_URL}"
  chmod +x ./kind
  sudo mv ./kind /usr/local/bin/kind

Install Helm:

  sudo snap install helm --classic

Allow the tupo user to run Docker commands without sudo:

  sudo usermod -aG docker tupo
  id tupo
  getent group docker

Log out and back in for the new Docker group membership to take effect. Confirm
Docker is installed and its daemon is running before continuing.

Check that the tools are available:

  kubectl version --client
  kind version
  helm version
  tilt version
  docker version


KUBERNETES CLUSTER AND CONFIG
=============================

kubectl and Tilt need a kubeconfig at ~/.kube/config. First check whether the
current user already has a working cluster configuration:

  kubectl config current-context
  kubectl get nodes

If these commands fail because the config is missing, choose one of the
following options.

Option A: Reuse a cluster configured for root
---------------------------------------------

Use this only if you trust the cluster and the root kubeconfig. A kubeconfig
contains credentials that may grant access to the cluster.

  mkdir -p ~/.kube
  sudo cp /root/.kube/config ~/.kube/config
  sudo chown -R tupo:tupo ~/.kube
  chmod 600 ~/.kube/config

Check that the copied config points to a cluster reachable by this user:

  kubectl config current-context
  kubectl get nodes

Option B: Create a local kind cluster
-------------------------------------

If there is no existing cluster to use, create a local cluster:

  kind create cluster --name tilt-go-demo
  kubectl config use-context kind-tilt-go-demo
  kubectl get nodes

The kind cluster uses Docker containers as Kubernetes nodes, so Docker must
be running and usable by the current user.


RUN THE PROJECT
===============

From the repository root, start Tilt:

  tilt up

Tilt will build the backend image, deploy PostgreSQL and the backend, and
forward these ports to the local machine:

  Backend:    http://localhost:8080/
  PostgreSQL: localhost:5432

The backend's root endpoint returns a JSON response. For example, check it
with:

  curl http://localhost:8080/

Keep Tilt running in the terminal while developing. Edit backend/main.go and
Tilt will sync Go source changes into the container, rebuild the server, and
restart the process. On startup, the backend creates the restaurant table and
seeds 100 demo restaurants if they have not already been seeded.

The Jenkins deployment installs K3s on the server when it is missing, then
deploys PostgreSQL and the backend with Helm. The backend is exposed on
NodePort 30080. Allow inbound TCP port 30080 in the cloud firewall, then access
the API at:

  http://<server-public-ip>:30080/api/restaurants

On hosts still using cgroup v1, deployment installs the Ubuntu 20.04 HWE
kernel, enables cgroup v2, and reboots once before installing K3s.

The API currently has no authentication. Public exposure also makes its
create, update, and delete endpoints accessible; do not expose it publicly
unless that is intended.


RESTAURANT API
==============

All API responses use JSON. The service validates request bodies and returns
structured errors. The health endpoint checks database connectivity:

  GET http://localhost:8080/healthz

List restaurants, with optional pagination and filters:

  GET /api/restaurants?limit=20&offset=0
  GET /api/restaurants?city=Portland
  GET /api/restaurants?cuisine=Italian
  GET /api/restaurants?q=spoon

The list response contains a data array and pagination metadata. The default
page size is 20 and the maximum is 100.

Retrieve one restaurant:

  GET /api/restaurants/{id}

Create a restaurant (price_level defaults to 2; rating defaults to 0):

  curl -X POST http://localhost:8080/api/restaurants \
    -H 'Content-Type: application/json' \
    -d '{
      "name": "Example Bistro",
      "cuisine": "French",
      "description": "A neighborhood bistro.",
      "address": "10 Main Street",
      "city": "Portland",
      "state": "Oregon",
      "postal_code": "97201",
      "country": "United States",
      "phone": "+1-555-555-0100",
      "email": "hello@example.com",
      "website": "https://example.com",
      "price_level": 2,
      "rating": 4.5
    }'

Update selected fields with PATCH:

  curl -X PATCH http://localhost:8080/api/restaurants/1 \
    -H 'Content-Type: application/json' \
    -d '{"rating": 4.8, "description": "Updated description."}'

Delete a restaurant:

  curl -i -X DELETE http://localhost:8080/api/restaurants/1

Restaurant fields are name, cuisine, description, address, city, state,
postal_code, country, phone, email, website, price_level (1-4), and rating
(0-5, up to two decimal places). Name, cuisine, address, city, and country are
required on creation.
Invalid input returns HTTP 400; missing IDs return HTTP 404. Creating returns
HTTP 201, successful reads/updates return HTTP 200, and successful deletes
return HTTP 204.

Stop Tilt with Ctrl-C, or from another terminal run:

  tilt down

To delete the local kind cluster when it is no longer needed:

  kind delete cluster --name tilt-go-demo


DATABASE DEVELOPMENT SETTINGS
==============================

The PostgreSQL chart uses these development defaults:

  Host:     postgres (inside Kubernetes)
  Port:     5432
  Database: tilt_demo
  User:     devuser
  Password: devpassword

These are local demo settings, not production credentials. Do not expose this
database or reuse these credentials in a production environment.


TROUBLESHOOTING
===============

Docker permission denied:
  Confirm the tupo user belongs to the docker group with "id tupo", then log
  out and back in. Confirm the Docker daemon is running with "docker version".

kubectl cannot connect to a cluster:
  Check "kubectl config current-context" and "kubectl get nodes". Ensure the
  selected context is the intended cluster and that its API server is
  reachable.

Backend or database is not ready:
  Open the Tilt UI shown by "tilt up" and inspect the resource logs. Check
  Kubernetes pods with "kubectl get pods".

Image build or live update fails:
  Run Tilt from the repository root. The backend Docker build context and the
  chart paths are relative to that directory.
