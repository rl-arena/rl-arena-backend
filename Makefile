.PHONY: build run test clean docker-build docker-up docker-down deps fmt lint

# 빌드
build:
	go build -o bin/server cmd/server/main.go

# 실행
run:
	go run cmd/server/main.go

# 테스트
test:
	go test -v ./...

# 클린
clean:
	rm -rf bin/

# Docker 빌드
docker-build:
	docker build -t rl-arena-backend .

# Docker Compose 시작
docker-up:
	docker-compose up -d

# Docker Compose 종료
docker-down:
	docker-compose down

# 의존성 설치
deps:
	go mod download
	go mod tidy

# 포맷
fmt:
	go fmt ./...

# 린트
lint:
	golangci-lint run