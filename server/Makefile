bindata:
	which bindata || { \
	  go get github.com/jteeuwen/go-bindata && \
	  cd $${GOPATH}/src/github.com/jteeuwen/go-bindata/go-bindata && \
	  go install .; \
	}

bindata.go: bindata
	@# If you want to change the output, just use -pkg and -o to change the package and destination file
	go-bindata ./pkg/web/today.html.template

test:
	test -e ./bindata.go && rm ./bindata.go
	go-bindata -debug ./today.html.template
	go test -v .

test-interactive:
	@test "$(test)" = "" && { \
		/bin/echo -e "\n***\nNeed test to run. Run \047make test=XYZ test-interactive\047\n***\n" >&2; \
		exit 1; \
	} || true
	rm bindata.go && go-bindata -debug ./today.html.template
	TIMETRACKER_INTERACTIVE_TESTS=on go test -v . -run "$(test)"
	rm test-db

bin: ./pkg/web/bindata.go
	go build ./server

.PHONY: \
	test \
	bin \

