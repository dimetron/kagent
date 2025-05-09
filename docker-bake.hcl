group "default" {
  targets = ["go", "python", "ui"]
}

buildx "docker-bake.hcl" {
  # This is the default builder
  name = "kagent-builder"
  # This is the default output
  output = "docker"
  # This is the default buildkit
  buildkit = "docker-container"
}

args = {
  LOCALPLATFORM= "arm64"
  TOOLS_GO_VERSION = "1.24.3",
  TOOLS_NODE_VERSION = "22",
  TOOLS_UV_VERSION = "0.7.2",
  TOOLS_K9S_VERSION = "0.50.4",
  TOOLS_KIND_VERSION = "0.27.0",
  TOOLS_ISTIO_VERSION = "1.25.2",
  TOOLS_KUBECTL_VERSION = "1.33.4"
  PROXY = "10.232.233.70:8080"
  DOCKER_REGISTRY = "illin4261.corp.amdocs.com:28090"
  DOCKER_REGISTRY_GOOGLE = "illin4261.corp.amdocs.com:28090"
}

target "go" {
  context = "./go"
  dockerfile = "Dockerfile"

  tags = ["controller:latest"]
}

target "python" {
  context = "./python"
  dockerfile = "Dockerfile"
  tags = ["app:latest"]
}

target "ui" {
  context = "./ui"
  dockerfile = "Dockerfile"
  tags = ["ui:latest"]
}