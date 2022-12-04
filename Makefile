build:
	protoc --go_out=paths=source_relative:. --twirp_out=paths=source_relative:. proto/service.proto
	docker build -t myko .

bash:
	docker run -it --entrypoint /bin/bash myko
