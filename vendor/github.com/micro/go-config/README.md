# Config [![GoDoc](https://godoc.org/github.com/micro/go-config?status.svg)](https://godoc.org/github.com/micro/go-config)

Go Config is a pluggable dynamic config library

Most config in applications are statically configured or include complex logic to load from multiple sources. 
Go-config makes this easy, pluggable and mergeable. You'll never have to deal with config in the same way again.

## Features

- **Dynamic** - load config on the fly as you need it
- **Pluggable** - choose which source to load from; file, env, consul
- **Mergeable** - merge and override multiple config sources
- **Fallback** - specify fallback values where keys don't exist
- **Watch** - Watch the config for changes

## Getting Started

- [Source](#source) - A backend from which config is loaded
- [Encoder](#encoder) - Handles encoding/decoding source config 
- [Reader](#reader) - Merges multiple encoded sources as a single format
- [Config](#config) - Config manager which manages multiple sources 
- [Usage](#usage) - Example usage of go-config
- [FAQ](#faq) - General questions and answers
- [TODO](#todo) - TODO tasks/features

## Sources

A `Source` is a backend from which config is loaded. Multiple sources can be used at the same time.

The following sources are supported:

- [configmap](https://github.com/micro/go-config/tree/master/source/configmap) - read from k8s configmap
- [consul](https://github.com/micro/go-config/tree/master/source/consul) - read from consul
- [etcd](https://github.com/micro/go-config/tree/master/source/etcd) - read from etcd v3
- [env](https://github.com/micro/go-config/tree/master/source/env) - read from environment variables
- [file](https://github.com/micro/go-config/tree/master/source/file) - read from file
- [flag](https://github.com/micro/go-config/tree/master/source/flag) - read from flags
- [grpc](https://github.com/micro/go-config/tree/master/source/grpc) - read from grpc server
- [memory](https://github.com/micro/go-config/tree/master/source/memory) - read from memory
- [microcli](https://github.com/micro/go-config/tree/master/source/microcli) - read from micro cli flags

TODO:

- vault
- git url

### ChangeSet

Sources return config as a ChangeSet. This is a single internal abstraction for multiple backends.

```go
type ChangeSet struct {
	// Raw encoded config data
	Data      []byte
	// MD5 checksum of the data
	Checksum  string
	// Encoding format e.g json, yaml, toml, xml
	Format    string
	// Source of the config e.g file, consul, etcd
	Source    string
	// Time of loading or update
	Timestamp time.Time
}
```

## Encoder

An `Encoder` handles source config encoding/decoding. Backend sources may store config in many different 
formats. Encoders give us the ability to handle any format. If an Encoder is not specified it defaults to json.

The following encoding formats are supported:

- json
- yaml
- toml
- xml
- hcl 

## Reader

A `Reader` represents multiple changesets as a single merged and queryable set of values.

```go
type Reader interface {
	// Merge multiple changeset into a single format
	Merge(...*source.ChangeSet) (*source.ChangeSet, error)
	// Return return Go assertable values
	Values(*source.ChangeSet) (Values, error)
	// Name of the reader e.g a json reader
	String() string
}
```

The reader makes use of Encoders to decode changesets into `map[string]interface{}` then merge them into 
a single changeset. It looks at the Format field to determine the Encoder. The changeset is then represented 
as a set of `Values` with the ability to retrive Go types and fallback where values cannot be loaded.

```go

// Values is returned by the reader
type Values interface {
	// Return raw data
        Bytes() []byte
	// Retrieve a value
        Get(path ...string) Value
	// Return values as a map
        Map() map[string]interface{}
	// Scan config into a Go type
        Scan(v interface{}) error
}
```

The `Value` interface allows casting/type asserting to go types with fallback defaults.

```go
type Value interface {
	Bool(def bool) bool
	Int(def int) int
	String(def string) string
	Float64(def float64) float64
	Duration(def time.Duration) time.Duration
	StringSlice(def []string) []string
	StringMap(def map[string]string) map[string]string
	Scan(val interface{}) error
	Bytes() []byte
}
```

## Config 

`Config` manages all config, abstracting away sources, encoders and the reader. 

It manages reading, syncing, watching from multiple backend sources and represents them as a single merged and queryable source.

```go

// Config is an interface abstraction for dynamic configuration
type Config interface {
        // provide the reader.Values interface
        reader.Values
	// Stop the config loader/watcher
	Close() error
	// Load config sources
	Load(source ...source.Source) error
	// Force a source changeset sync
	Sync() error
	// Watch a value for changes
	Watch(path ...string) (Watcher, error)
}
```

## Usage

- [Sample Config](#sample-config)
- [New Config](#new-config)
- [Load File](#load-file)
- [Read Config](#read-config)
- [Read Values](#read-values)
- [Watch Path](#watch-path)
- [Multiple Sources](#merge-sources)
- [Set Source Encoder](#set-source-encoder)
- [Add Reader Encoder](#add-reader-encoder)



### Sample Config

A config file can be of any format as long as we have an Encoder to support it.

Example json config:

```json
{
    "hosts": {
        "database": {
            "address": "10.0.0.1",
            "port": 3306
        },
        "cache": {
            "address": "10.0.0.2",
            "port": 6379
        }
    }
}
```

### New Config

Create a new config (or just make use of the default instance)

```go
import "github.com/micro/go-config"

conf := config.NewConfig()
```

### Load File

Load config from a file source. It uses the file extension to determine config format.

```go
import (
	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
)

// Load json config file
config.Load(file.NewSource(
	file.WithPath("/tmp/config.json"),
))
```

Load a yaml, toml or xml file by specifying a file with the appropriate file extension

```go
// Load yaml config file
config.Load(file.NewSource(
        file.WithPath("/tmp/config.yaml"),
))
```

If an extension does not exist, specify the encoder

```go
enc := toml.NewEncoder()

// Load toml file with encoder
config.Load(file.NewSource(
        file.WithPath("/tmp/config"),
	source.WithEncoder(enc),
))
```

### Read Config

Read the entire config as a map

```go
// retrieve map[string]interface{}
conf := config.Map()

// map[cache:map[address:10.0.0.2 port:6379] database:map[address:10.0.0.1 port:3306]]
fmt.Println(conf["hosts"])
```

Scan the config into a struct

```go
type Host struct {
        Address string `json:"address"`
        Port int `json:"port"`
}

type Config struct{
	Hosts map[string]Host `json:"hosts"`
}

var conf Config

config.Scan(&conf)

// 10.0.0.1 3306
fmt.Println(conf.Hosts["database"].Address, conf.Hosts["database"].Port)
```

### Read Values

Scan a value from the config into a struct

```go
type Host struct {
	Address string `json:"address"`
	Port int `json:"port"`
}

var host Host

config.Get("hosts", "database").Scan(&host)

// 10.0.0.1 3306
fmt.Println(host.Address, host.Port)
```

Read individual values as Go types

```go
// Get address. Set default to localhost as fallback
address := config.Get("hosts", "database", "address").String("localhost")

// Get port. Set default to 3000 as fallback
port := config.Get("hosts", "database", "port").Int(3000)
```

### Watch Path

Watch a path for changes. When the file changes the new value will be made available.

```go
w, err := config.Watch("hosts", "database")
if err != nil {
	// do something
}

// wait for next value
v, err := w.Next()
if err != nil {
	// do something
}

var host Host

v.Scan(&host)
```

### Multiple Sources

Multiple sources can be loaded and merged. Merging priority is in reverse order. 

```go
config.Load(
	// base config from env
	env.NewSource(),
	// override env with flags
	flag.NewSource(),
	// override flags with file
	file.NewSource(
		file.WithPath("/tmp/config.json"),
	),
)
```

### Set Source Encoder

A source requires an encoder to encode/decode data and specify the changeset format.

The default encoder is json. To change the encoder to yaml, xml, toml specify as an option.

```go
e := yaml.NewEncoder()

s := consul.NewSource(
	source.WithEncoder(e),
)
```

### Add Reader Encoder

The reader uses encoders to decode data from sources with different formats.

The default reader supports json, yaml, xml, toml and hcl. It represents the merged config as json.

Add a new encoder by specifying it as an option.

```go
e := yaml.NewEncoder()

r := json.NewReader(
	reader.WithEncoder(e),
)
```

## FAQ

### How is this different from Viper?

[Viper](https://github.com/spf13/viper) and go-config are solving the same problem. Go-config provides a different interface 
and is part of the larger micro ecosystem of tooling.

### What's the difference between Encoder and Reader?

The encoder is used by a backend source to encode/decode it's data. The reader uses encoders to decode data from multiple 
sources with different formats, it then merges them into a single encoding format. 

In the case of a file source , we use the file extension to determine the config format so the encoder is not used. 

In the case of consul, etcd or similar key-value source we may load from a prefix containing multiple keys which means 
the source needs to understand the encoding so it can return a single changeset. 

In the case of environment variables and flags we also need a way to encode the values as bytes and specify the format so 
it can later be merged by the reader.

### Why is changeset data not represented as map[string]interface{}?

In some cases source data may not actually be key-value so it's easier to represent it as bytes and defer decoding to 
the reader.

## TODO

- Config Versioning - version the internal changeset and allow the user to specify version
- Canarying/Coordination - allow coordination of instances that subscribe to config and canary an update
