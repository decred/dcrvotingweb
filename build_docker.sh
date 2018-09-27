#!/bin/bash -e

# Build hardforkdemo
rm -rf docker_context
mkdir docker_context

docker run --rm -it -v $(pwd):/src decred/decred-golang-builder-1.11 /bin/bash -c "\
  rsync -ra --filter=':- .gitignore'  \
  /src/ /go/src/github.com/decred/hardforkdemo/ && \
  cd github.com/decred/hardforkdemo/ && \
  env GO111MODULE=on go install && \
  cp /go/bin/hardforkdemo /src/docker_context/hardforkdemo
"

# Build docker container
cp -r public/ docker_context/public

docker build . \
	-t hardforkdemo \
	-f ./Dockerfile;

# Clean up
rm -rf docker_context

echo ""
echo "==================="
echo "  Build complete"
echo "==================="
echo ""
echo "You can now run hardforkdemo with the following command:"
echo "    docker run -it -v ~/.dcrd:/root/.dcrd -v ~/.hardforkdemo:/root/.hardforkdemo -p <local port>:8000 hardforkdemo:latest"
echo ""
