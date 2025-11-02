docker volume create kanidmd
docker create --name kanidmd \
  -p '443:8443' \
  -p '636:3636' \
  -v kanidmd:/data \
  docker.io/kanidm/server:latest

docker cp server.toml kanidmd:/data/server.toml