# Migrating from stid

This library was previously known as `stid`, before getting renamed to `flexid`.
The last version released under `stid` and module path (`github.com/amterp/stid`) was v1.2.0.
The flexid library, using the module path `github.com/amterp/flexid`, begins its versioning at v1.3.0,
continuing semantically from the previous version.

To migrate to flexid:

1. `go get github.com/amterp/flexid`
2. Find replace `"github.com/amterp/stid"` with `fid "github.com/amterp/flexid"` in your Go files.
3. Find replace `stid.` references with `fid.` in your Go files.
4. After removing all references to stid, run `go mod tidy` and it should remove stid from your go.mod and go.sum.
