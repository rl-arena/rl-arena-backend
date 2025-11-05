FROM golang:1.25-alpine AS builder

WORKDIR /app

# 의존성 복사 및 다운로드
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사
COPY . .

# 빌드
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server cmd/server/main.go

# 실행 단계
FROM alpine:latest

WORKDIR /app

# 필요한 패키지 설치
RUN apk --no-cache add ca-certificates tzdata

# 빌더에서 바이너리 복사
COPY --from=builder /app/bin/server /app/server

# 환경변수 설정
ENV TZ=Asia/Seoul

EXPOSE 8080

CMD ["/app/server"]