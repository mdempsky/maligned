# Maligned

A tool to detect Go structs that would take less memory if their fields were optimally ordered sorted. Confused? Read this article for a good explanation of the problem this solves: https://medium.com/@felipedutratine/how-to-organize-the-go-struct-in-order-to-save-memory-c78afcf59ec2

Install: `go get -u github.com/ValarDragon/maligned`

Usage:
By default, maligned just tells you that an improvement is possible. (When applicable) If you add the [-v] flag, it prints out the order of the field names that you should use in order to optimize its memory usage. Maligned needs to be pointed at an entire package, as it needs to know the size of all the relevant structs. Currently it looks for the package name you point in a path originating from the $GOPATH, and $GOBIN.

Command usage: `maligned [-v] <package1> <package1> ...`

In `server/init.go`, there exists the following struct:
```
// Storage for init command input parameters
type InitConfig struct {
	ChainID   string
	GenTxs    bool
	GenTxsDir string
	Overwrite bool
}
```

Sample output:
```
$ cd cosmos/cosmos-sdk
$ maligned ./...
go/src/github.com/cosmos/cosmos-sdk/server/init.go:58:17: struct of size 48 could be 40
go/src/github.com/cosmos/cosmos-sdk/store/iavlstore.go:194:19: struct of size 152 could be 144
go/src/github.com/cosmos/cosmos-sdk/x/stake/validator.go:21:16: struct of size 608 could be 600
go/src/github.com/cosmos/cosmos-sdk/x/stake/client/rest/query.go:84:27: struct of size 600 could be 592

$ maligned -v github.com/cosmos/cosmos-sdk/server
go/src/github.com/cosmos/cosmos-sdk/server/init.go:58:17: struct of size 48 could be 40
Reorder struct as:
ChainID
GenTxsDir
GenTxs
Overwrite
```
