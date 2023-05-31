.PHONY: build run test

build:
	docker run --rm -v $(PWD):/go/src/go-filter -w /go/src/go-filter \
		-e GOPROXY=https://goproxy.cn \
		golang:1.19 \
		go build -v -o libgolang.so -buildmode=c-shared -buildvcs=false .

test-bind-mode:
	docker run --rm -v $(PWD)/example/envoy.yaml:/etc/envoy/envoy.yaml \
		-v $(PWD)/libgolang.so:/etc/envoy/libgolang.so \
		-e GODEBUG=cgocheck=0 \
		-p 10000:10000 \
		envoyproxy/envoy:contrib-dev \
		envoy -c /etc/envoy/envoy.yaml &
	sleep 5
	go test -v -tags cgo test/e2e_bind_test.go

test-search-mode:
	docker run --rm -v $(PWD)/example/envoy-search.yaml:/etc/envoy/envoy.yaml \
		-v $(PWD)/libgolang.so:/etc/envoy/libgolang.so \
		-e GODEBUG=cgocheck=0 \
		-p 10000:10000 \
		envoyproxy/envoy:contrib-dev \
		envoy -c /etc/envoy/envoy.yaml &
	sleep 5
	go test -v -tags cgo test/e2e_search_test.go