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
Tilt will sync the file into the container, rebuild the Go server, and restart
the process.

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
