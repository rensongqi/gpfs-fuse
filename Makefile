.PHONY: build

BUILD_TIME=$(shell date +%F-%Z-%H%M%S)

build:
	- git pull;docker build -t gpfs-fuse:$(BUILD_TIME) .;docker push gpfs-fuse:$(BUILD_TIME)