# gRPC Client Implementation Guide

이 문서는 Backend에 gRPC 클라이언트를 구현하는 과정을 설명합니다.

## 1. Proto 파일 준비

Proto 파일이 `proto/executor.proto`에 복사되었습니다.

## 2. Go 코드 생성

### 필요한 도구 설치

```bash
# protoc-gen-go 설치 (이미 완료)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# protoc-gen-go-grpc 설치 (이미 완료)  
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### protoc 설치 (Windows)

1. https://github.com/protocolbuffers/protobuf/releases 에서 최신 버전 다운로드
2. `protoc-<version>-win64.zip` 다운로드
3. 압축 해제 후 `protoc.exe`를 PATH에 추가

또는 Chocolatey 사용:
```bash
choco install protoc
```

### Proto 컴파일

```bash
# pkg/pb 디렉토리에 Go 코드 생성
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/executor.proto
```

생성될 파일:
- `proto/executor.pb.go` - Proto 메시지 정의
- `proto/executor_grpc.pb.go` - gRPC 클라이언트/서버 코드

## 3. gRPC 클라이언트 구현

`pkg/executor/client.go`를 HTTP에서 gRPC로 전환합니다.

### 변경 사항

**Before (HTTP):**
```go
httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
```

**After (gRPC):**
```go
conn, err := grpc.Dial(address, grpc.WithInsecure())
client := pb.NewExecutorClient(conn)
response, err := client.RunMatch(ctx, request)
```

### K8s Service DNS

프로덕션 환경에서는 K8s Service DNS를 사용합니다:
```go
address := "rl-arena-executor.rl-arena.svc.cluster.local:50051"
```

로컬 개발 환경:
```go
address := "localhost:50051"
```

## 4. 다음 단계

1. protoc 설치
2. Proto 컴파일
3. `pkg/executor/client.go` 수정
4. gRPC 의존성 추가 (`go.mod`)
5. 테스트

## 참고

- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [Protocol Buffers Go Tutorial](https://protobuf.dev/getting-started/gotutorial/)
