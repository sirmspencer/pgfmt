// learn: go.mod defines a module: a versioned set of packages. the module path is both its name and
// learn: where `go install` downloads it from. every import of this project's packages starts with
// learn: this path.
module github.com/sirmspencer/pgfmt

// learn: the minimum go version the code needs. newer toolchains still build it.
go 1.26.4

// learn: dependencies with exact versions. `go get` adds them, `go mod tidy` removes unused ones.
// learn: go.sum (next to this file) stores checksums so builds are reproducible and tamper-evident.
require (
	github.com/pganalyze/pg_query_go/v6 v6.2.5
	google.golang.org/protobuf v1.33.0
)
