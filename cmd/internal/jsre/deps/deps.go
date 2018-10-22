package deps

//go:generate go-bindata -nometadata -pkg deps -o bindata.go polyfill.js vite.js docs.js
//go:generate gofmt -w -s bindata.go
