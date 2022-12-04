module github.com/mykodev/myko

go 1.19

require (
	github.com/gocql/gocql v1.2.1
	github.com/twitchtv/twirp v8.1.3+incompatible
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/golang/snappy v0.0.3 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

replace github.com/gocql/gocql v1.2.1 => github.com/scylladb/gocql v1.7.2
