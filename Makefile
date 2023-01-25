build:
	protoc --go_out=paths=source_relative:. --twirp_out=paths=source_relative:. proto/service.proto
	docker build -t myko .

dev:
	go run ./cmd/myko -config /tmp/myko.yaml

benchmark-ingest:
	go run ./benchmarks/*.go -n 2000 -events 200

bash:
	docker run -it --entrypoint /bin/bash myko

ecr-push: build
	docker tag myko:latest public.ecr.aws/q1p8v8z2/myko:latest
	docker push public.ecr.aws/q1p8v8z2/myko:latest

ecr-login:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws/q1p8v8z2
