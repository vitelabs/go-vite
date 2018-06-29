### algorithm
* AES: AES-256-GCM

* KDF: scrypt


### KeyStore Filename 
`DirPath + "/v-i-t-e-" + hex(addr)`

### Default Dir Path
```
func DefaultDataDir() string {
	home := homeDir()
	if home != "" {
		return filepath.Join(home, ".viteisbest")
	}
	return ""
}
```
