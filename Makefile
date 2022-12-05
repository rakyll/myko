build:
	protoc --go_out=paths=source_relative:. --twirp_out=paths=source_relative:. proto/service.proto
	docker build -t myko .

bash:
	docker run -it --entrypoint /bin/bash myko

ecr-push: build
	docker tag myko:latest public.ecr.aws/q1p8v8z2/myko:latest
	docker push public.ecr.aws/q1p8v8z2/myko:latest

ecr-login:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws/q1p8v8z2
