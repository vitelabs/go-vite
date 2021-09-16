# Build

```
cd `go env GOPATH`/src/github.com/vitelabs/go-vite

git checkout ${version}

docker build -t vitelabs/gvite:${version} -f docker/Dockerfile .
```

# Run With Docker

```
# run with a docker container
docker run --log-opt max-size=100m --name="gvite-node" -v $HOME/.gvite/:/root/.gvite/ -p 48132:48132 -p 41420:41420 -p 8483:8483 -p 8484:8484 -p 8483:8483/udp -d vitelabs/gvite:${version}


# check vite log
docker logs --tail 100 -f gvite-node
```
