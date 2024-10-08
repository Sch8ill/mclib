bin_name=mcli
target=cmd/main.go

all: build

run:
	go run $(target)

build:
	go build -ldflags="-s" -trimpath -o build/$(bin_name) $(target)

multi-arch:
	scripts/build-multi-arch.sh $(target) build/$(bin_name)

clean:
	rm -rf build