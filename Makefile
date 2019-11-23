.PHONY: build push

build:
	docker build -t koobz/nodalingresser .

push: build
	docker push koobz/nodalingresser
