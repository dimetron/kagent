#!/usr/bin/env bash

export BUILDKIT_VERSION=v0.22.0
export OS=$(uname -o)

export script_dir=$(dirname "$0")

echo "BuildKit version: $BUILDKIT_VERSION"
pwd
cat $script_dir/buildkitd.toml
echo "-----"

# in rancher we need share volume with docker.sock
if [ "${OS}" == "Darwin" ] ; then
  echo "Running on Mac"
  rdctl shell sudo chmod 777 /var/run/docker.sock || true
  timeout 10 sudo chmod 777 /var/run/docker.sock  || true
fi

#make sure no osxauth is present
mkdir -p $HOME/.docker
cat <<EOT > $HOME/.docker/config.json
{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "ZGltZXRyb246ZGNrcl9wYXRfaVoycmhOOGdFNlVoRE1HNUgxSmtYcWdkbllj"
			},
		"illin4261.corp.amdocs.com:28090": {
			"auth": "bnRzZGVwbG95bWVudDpudHNkZXBsb3ltZW50QDIwMjM="
		},
		"illin4261.corp.amdocs.com:8084": {
			"auth": "bnRzZGVwbG95bWVudDpudHNkZXBsb3ltZW50QDIwMjM="
		},
		"illin4261.corp.amdocs.com:6000": {
			"auth": "bnRzZGVwbG95bWVudDpudHNkZXBsb3ltZW50QDIwMjM="
		},
		"illin5225.corp.amdocs.com:5000": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"illin5225.corp.amdocs.com:5002": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"illin5225.corp.amdocs.com:7000": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"illin5564.corp.amdocs.com:5000": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"illin5564.corp.amdocs.com:7000": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"illin5638.corp.amdocs.com:15005": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"illin5638.corp.amdocs.com:5002": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"illin5638.corp.amdocs.com:15006": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"illin5638.corp.amdocs.com:5005": {
			"auth": "cHNtZG9ja2VyOnVuaXgxMQ=="
		},
		"ilnexusdmz02.corp.amdocs.com:8084": {
			"auth": "bnRzZGVwbG95bWVudDpudHNkZXBsb3ltZW50QDIwMjM="
		},
		"ilnexusdmz02.corp.amdocs.com:8091": {
			"auth": "bnRzZGVwbG95bWVudDpudHNkZXBsb3ltZW50QDIwMjM="
		},
		"artifactory-isr.corp.amdocs.com" : {
      "auth": "bXMzNjBfc2FfY2lAYW1kb2NzLmNvbTpjbVZtZEd0dU9qQXhPakUzTlRRM016SXhPVEU2Vld4clVrdHRiVlZuYmxSMGFXVTBjM1pyZGxsTFZuTm9iVVYx"
    },
		"artifactory.corp.amdocs.com" : {
      "auth": "bXMzNjBfc2FfY2lAYW1kb2NzLmNvbTpjbVZtZEd0dU9qQXhPakUzTlRRM016SXhPVEU2Vld4clVrdHRiVlZuYmxSMGFXVTBjM1pyZGxsTFZuTm9iVVYx"
    },
		"localhost:6000": {},
		"registry:6001": {}
	}
}
EOT

docker image inspect moby/buildkit:$BUILDKIT_VERSION  > /dev/null 2>&1 || docker pull docker-registry-proxy.corp.amdocs.com/moby/buildkit:$BUILDKIT_VERSION
docker image inspect moby/buildkit:$BUILDKIT_VERSION  > /dev/null 2>&1 || docker tag  docker-registry-proxy.corp.amdocs.com/moby/buildkit:$BUILDKIT_VERSION moby/buildkit:$BUILDKIT_VERSION

#make sure qemu-user-static installed
if [[ "$(uname -m)"  == "x86_64" ]]; then
  docker image inspect multiarch/qemu-user-static:latest 2>&1 > /dev/null || docker pull docker-registry-proxy.corp.amdocs.com/multiarch/qemu-user-static:latest
  docker image inspect multiarch/qemu-user-static:latest 2>&1 > /dev/null || rdctl shell docker tag docker-registry-proxy.corp.amdocs.com/multiarch/qemu-user-static:latest multiarch/qemu-user-static:latest
  docker run --rm --privileged multiarch/qemu-user-static --reset -p yes || true
  #docker run --rm --privileged docker tag docker-registry-proxy.corp.amdocs.com/multiarch/qemu-user-static:latest --reset -p yes
fi

export BUILDX_NAME=kagent-builder
echo "Activate BuildX -> $BUILDX_NAME with buildkit version: $BUILDKIT_VERSION"
#docker buildx rm $BUILDX_NAME || true
docker network inspect infra || docker network create infra
docker buildx inspect $BUILDX_NAME > /dev/null 2>&1  || docker buildx create --driver-opt "network=host,image=docker-registry-proxy.corp.amdocs.com/moby/buildkit:$BUILDKIT_VERSION" --config $script_dir/buildkitd.toml --name $BUILDX_NAME --use
docker buildx use $BUILDX_NAME || true
docker buildx inspect $BUILDX_NAME

echo "Infra network containers:"
docker network inspect infra | jq '.[].Containers[].Name' -r
echo "--- buildx containers configured ---"
