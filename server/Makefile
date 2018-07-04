test:
	cd ../webui && make DEBUG=true bindata
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

