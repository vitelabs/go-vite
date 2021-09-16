# Build

```
cd `go env GOPATH`/src/github.com/vitelabs/go-vite

git checkout ${version}

docker build -t vitelabs/gvite:${version} -f docker/Dockerfile .

```

# Run With Docker

```
docker run --log-opt max-size=100m -v $HOME/.gvite/:/root/.gvite/ -p 48132:48132 -p 41420:41420 -p 8483:8483 -p 8484:8484 -p 8483:8483/udp -d vitelabs/gvite:${version}
```
