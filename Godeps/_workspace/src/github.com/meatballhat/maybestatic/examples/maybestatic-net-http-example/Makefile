.PHONY: all
all: clean maybestatic-net-http-example

.PHONY: clean
clean:
	rm -vf maybestatic-net-http-example

maybestatic-net-http-example:
	go build -x ./...

bindata.go: .go-bindata-bootstrap $(wildcard public/*)
	go-bindata -prefix=public -o=$@ ./public

.go-bindata-bootstrap:
	go get -x github.com/jteeuwen/go-bindata > $@
