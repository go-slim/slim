.PHONY: help run-cors run-rate-limiter run-logger-recovery run-nego run-static \
	build-cors build-rate-limiter build-logger-recovery build-nego build-static \
	docker-cors docker-rate-limiter docker-logger-recovery docker-nego docker-static

help:
	@echo "Targets:"
	@echo "  run-{cors,rate-limiter,logger-recovery,nego,static}      Run example locally"
	@echo "  build-{cors,rate-limiter,logger-recovery,nego,static}    Build example binary"
	@echo "  docker-{cors,rate-limiter,logger-recovery,nego,static}   Build+Run in Docker"

# ----------------
# Local run
# ----------------
run-cors:
	go run ../../slim/examples/cors

run-rate-limiter:
	go run ../../slim/examples/rate-limiter

run-logger-recovery:
	go run ../../slim/examples/logger-recovery

run-nego:
	go run ../../slim/examples/nego

run-static:
	go run ../../slim/examples/static

# ----------------
# Local build
# ----------------
build-cors:
	go build -o ./bin/cors ../../slim/examples/cors

build-rate-limiter:
	go build -o ./bin/rate-limiter ../../slim/examples/rate-limiter

build-logger-recovery:
	go build -o ./bin/logger-recovery ../../slim/examples/logger-recovery

build-nego:
	go build -o ./bin/nego ../../slim/examples/nego

build-static:
	go build -o ./bin/static ../../slim/examples/static

# ----------------
# Dockerized run (build + run)
# ----------------
docker-cors:
	docker build -f ../../slim/examples/Dockerfile --build-arg EXAMPLE=slim/examples/cors -t slim-example-cors ../..
	docker run --rm -p 1325:1325 slim-example-cors

docker-rate-limiter:
	docker build -f ../../slim/examples/Dockerfile --build-arg EXAMPLE=slim/examples/rate-limiter -t slim-example-rate-limiter ../..
	docker run --rm -p 1326:1326 slim-example-rate-limiter

docker-logger-recovery:
	docker build -f ../../slim/examples/Dockerfile --build-arg EXAMPLE=slim/examples/logger-recovery -t slim-example-logger-recovery ../..
	docker run --rm -p 1327:1327 slim-example-logger-recovery

docker-nego:
	docker build -f ../../slim/examples/Dockerfile --build-arg EXAMPLE=slim/examples/nego -t slim-example-nego ../..
	docker run --rm -p 1328:1328 slim-example-nego

docker-static:
	docker build -f ../../slim/examples/Dockerfile --build-arg EXAMPLE=slim/examples/static -t slim-example-static ../..
	docker run --rm -p 1329:1329 -v $$(pwd)/../../public:/app/public slim-example-static
