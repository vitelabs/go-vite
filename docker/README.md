

# Build

```
cd `go env GOPATH`/src/github.com/vitelabs/go-vite

docker build -t vitelabs/gvite:test -f docker/Dockerfile .

```


# Run With Docker

```
docker run -v $HOME/.gvite/:/root/.gvite/ -p 48132:48132 -p 41420:41420 -p 8483:8483 -p 8484:8484 -p 8483:8483/udp -d vitelabs/gvite:test
```



# quickly Build from local binary

```
gvite-linux
docker build -t vitelabs/gvite-nightly:test . -f docker/Dockerfile.preBuild
```