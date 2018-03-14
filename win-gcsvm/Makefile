GO_COMPILE=linuxkit/go-compile:bb0c6ae2f12a1b55df24ebce2067784a931285df

.PHONY: gcsvm.exe
gcsvm.exe: gcsvm.go
	docker run -it --rm \
		-v $(CURDIR):/go/src/github.com/rn/utils/gcsvm \
		-w /go/src/github.com/rn/utils/gcsvm \
		-e GOOS=windows \
		--entrypoint go $(GO_COMPILE) build gcsvm.go

.PHONY: vendor
vendor:
	docker run -it --rm \
		-v $(CURDIR):/go/src/github.com/rn/utils/gcsvm \
		-w /go/src/github.com/rn/utils/gcsvm \
		--entrypoint /go/bin/vndr $(GO_COMPILE)

.PHONY: clean
clean:
	rm gcsvm.exe
