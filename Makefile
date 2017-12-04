bindata.go:
	which bindata || { \
	  go get github.com/jteeuwen/go-bindata && \
	  cd $${GOPATH}/src/github.com/jteeuwen/go-bindata/go-bindata && \
	  go install .; \
	}
	go-bindata ./today.html.template

all: bindata.go
	go build .

