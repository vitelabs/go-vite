package deps

//go:generate go-bindata -nometadata -pkg deps -o bindata.go bignumber.js vite.js typedarray.js
//go:generate gofmt -w -s bindata.go
