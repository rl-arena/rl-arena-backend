# Phase 1 완료 - gRPC 및 Docker 빌드 파이프라인

## 개요
Phase 1에서는 Backend와 Executor 간의 통신을 HTTP에서 gRPC로 전환하고, Agent 제출 시 Kaniko를 사용한 Docker 이미지 빌드 파이프라인을 구축했습니다.

## 완료된 작업

### 1. gRPC 클라이언트 구현 ✅
- **Proto 파일 수정**
  - `proto/executor.proto`에 `go_package` 옵션 추가
  - Go 코드 생성 활성화

- **Proto 컴파일**
  ```bash
  protoc --go_out=. --go-grpc_out=. proto/executor.proto
  ```
  - 생성된 파일: `proto/executor.pb.go`, `proto/executor_grpc.pb.go`

- **Client 완전 재작성**
  - `pkg/executor/client.go`: HTTP → gRPC 전환
  - K8s Service DNS 지원: `rl-arena-executor.rl-arena.svc.cluster.local:50051`
  - 주요 변경사항:
    ```go
    // 이전 (HTTP)
    func NewClient(baseURL string) *Client

    // 현재 (gRPC)
    func NewClient(address string) (*Client, error)
    func (c *Client) Close() error
    func (c *Client) ExecuteMatch(req ExecuteMatchRequest) (*ExecuteMatchResponse, error)
    func (c *Client) HealthCheck() error
    ```

- **의존성 추가**
  - `google.golang.org/grpc v1.76.0`
  - `google.golang.org/protobuf v1.36.10`

- **Router 업데이트**
  - `internal/api/router.go`: NewClient 에러 처리 추가

### 2. Submission 모델에 Docker 이미지 필드 추가 ✅
- **모델 업데이트**
  - `internal/models/submission.go`에 3개 필드 추가:
    ```go
    DockerImageURL *string `json:"dockerImageUrl,omitempty" db:"docker_image_url"`
    BuildJobName   *string `json:"buildJobName,omitempty" db:"build_job_name"`
    BuildPodName   *string `json:"buildPodName,omitempty" db:"build_pod_name"`
    ```

- **DB 마이그레이션**
  - `migrations/003_add_docker_image.sql` 생성
  - 3개 컬럼 추가 및 2개 인덱스 생성
  - 적용 방법:
    ```bash
    psql -U postgres -d rl_arena < migrations/003_add_docker_image.sql
    ```

- **Repository 업데이트**
  - `internal/repository/submission_repository.go` 모든 메서드 수정:
    - `Create()`: INSERT 쿼리 및 Scan 업데이트
    - `FindByID()`: SELECT 및 Scan 업데이트
    - `FindByAgentID()`: SELECT 및 Scan 업데이트
    - `GetActiveSubmission()`: SELECT 및 Scan 업데이트

### 3. Kaniko Build Pipeline 구현 ✅
- **BuilderService 생성**
  - `internal/service/builder_service.go` (247 lines)
  - 주요 기능:
    - `BuildAgentImage()`: Kaniko Job 생성 및 Docker 이미지 빌드
    - `createKanikoJob()`: K8s Job YAML 동적 생성
    - `GetBuildStatus()`: 빌드 상태 확인
    - `GetBuildLogs()`: Pod 로그 조회
    - `waitForPod()`: Pod 생성 대기

- **Kaniko Job 구조**
  ```yaml
  InitContainer (git-clone):
    - Git 저장소 클론 → /workspace
  
  Container (kaniko):
    - Dockerfile 빌드
    - 이미지 푸시: docker.io/<registry>/<agentId>:<submissionId>-v<version>
    - 레지스트리 인증: registry-credentials Secret
  ```

- **K8s 클라이언트 의존성**
  - `k8s.io/client-go v0.34.1`
  - `k8s.io/api v0.34.1`
  - `k8s.io/apimachinery v0.34.1`

- **SubmissionService 통합**
  - `internal/service/service.go`: BuilderService 주입
  - `CreateFromURL()`: TODO 주석으로 빌드 호출 위치 표시
  - 비동기 빌드 지원 준비

- **Router 통합**
  - `internal/api/router.go`: BuilderService 초기화
  - K8s 환경에서만 활성화 (`cfg.UseK8s == true`)

### 4. Container Registry 설정 ✅
- **Secret YAML**
  - `k8s/registry-secret.yaml` 생성
  - Docker Registry 인증 정보 저장
  - 생성 방법 주석 포함:
    ```bash
    kubectl create secret docker-registry registry-credentials \
      --docker-server=https://index.docker.io/v1/ \
      --docker-username=<USERNAME> \
      --docker-password=<PASSWORD> \
      --docker-email=<EMAIL> \
      --namespace=rl-arena
    ```

- **ConfigMap 업데이트**
  - `k8s/configmap.yaml`:
    ```yaml
    CONTAINER_REGISTRY_URL: "docker.io/your-registry"
    CONTAINER_REGISTRY_SECRET: "registry-credentials"
    K8S_NAMESPACE: "rl-arena"
    USE_K8S: "true"
    ```

- **Config 구조체**
  - `internal/config/config.go`:
    ```go
    K8sNamespace             string
    UseK8s                   bool
    ContainerRegistryURL     string
    ContainerRegistrySecret  string
    ```

## 아키텍처 변경사항

### Before (Phase 0)
```
Backend (HTTP) → Executor (HTTP)
Submission: code_url (GitHub 링크만)
```

### After (Phase 1)
```
Backend (gRPC) → Executor (gRPC)
         ↓
   BuilderService
         ↓
   Kaniko Job (K8s)
         ↓
   Container Registry
         ↓
Submission: code_url + docker_image_url + build_job_name + build_pod_name
```

## 주요 파일 목록

### 새로 생성된 파일
1. `proto/executor.pb.go` - Proto 컴파일 결과 (Go)
2. `proto/executor_grpc.pb.go` - gRPC 서비스 인터페이스
3. `internal/service/builder_service.go` - Kaniko 빌드 서비스
4. `k8s/registry-secret.yaml` - 레지스트리 인증 Secret
5. `migrations/003_add_docker_image.sql` - DB 스키마 업데이트

### 수정된 파일
1. `proto/executor.proto` - go_package 옵션 추가
2. `pkg/executor/client.go` - HTTP → gRPC 완전 재작성
3. `internal/models/submission.go` - Docker 필드 3개 추가
4. `internal/repository/submission_repository.go` - 모든 쿼리 업데이트
5. `internal/service/service.go` - BuilderService 통합
6. `internal/api/router.go` - BuilderService 초기화
7. `internal/config/config.go` - K8s/Registry 설정 추가
8. `k8s/configmap.yaml` - 레지스트리 설정 추가
9. `go.mod` - gRPC, K8s 의존성 추가

## 빌드 검증

```bash
# Proto 컴파일
protoc --go_out=. --go-grpc_out=. proto/executor.proto  ✅

# Go 빌드
go mod tidy          ✅
go build ./...       ✅

# 에러 없음, 모든 패키지 컴파일 성공
```

## 배포 가이드

### 1. DB 마이그레이션 실행
```bash
psql -U postgres -d rl_arena -f migrations/003_add_docker_image.sql
```

### 2. Container Registry Secret 생성
```bash
# Docker Hub 예시
kubectl create secret docker-registry registry-credentials \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=myusername \
  --docker-password=mypassword \
  --docker-email=myemail@example.com \
  --namespace=rl-arena
```

### 3. ConfigMap 업데이트
```bash
# k8s/configmap.yaml에서 CONTAINER_REGISTRY_URL 수정
# 예: docker.io/myusername

kubectl apply -f k8s/configmap.yaml
```

### 4. Backend 재배포
```bash
kubectl apply -f k8s/deployment.yaml
kubectl rollout restart deployment/rl-arena-backend -n rl-arena
```

## 테스트 시나리오

### 1. gRPC 통신 테스트
```bash
# Backend 로그 확인
kubectl logs -f deployment/rl-arena-backend -n rl-arena

# Executor와 gRPC 연결 성공 확인
# "Connected to executor: rl-arena-executor.rl-arena.svc.cluster.local:50051"
```

### 2. Docker 이미지 빌드 테스트
```bash
# Agent 제출 (GitHub URL 방식)
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "agentId": "agent-123",
    "codeUrl": "https://github.com/username/repo.git"
  }'

# Kaniko Job 생성 확인
kubectl get jobs -n rl-arena | grep build-

# Pod 로그 확인
kubectl logs -f pod/build-<submission-id>-<hash> -n rl-arena -c kaniko

# Submission docker_image_url 확인
psql -U postgres -d rl_arena -c "SELECT docker_image_url FROM submissions WHERE id='<submission-id>';"
```

### 3. 이미지 Pull 테스트
```bash
# Executor가 빌드된 이미지 사용 가능한지 확인
kubectl run test-pull --image=<docker_image_url> --rm -it --restart=Never -n rl-arena
```

## 다음 단계 (Phase 2)

### 1. 빌드 모니터링 및 상태 업데이트
- Kaniko Job 완료 감지
- Submission 상태 자동 업데이트 (building → success/failed)
- 빌드 로그를 DB에 저장

### 2. 빌드 재시도 및 오류 처리
- 빌드 실패 시 자동 재시도 (backoff)
- 빌드 오류 메시지 사용자에게 표시
- 빌드 타임아웃 처리

### 3. Match 실행 시 Docker 이미지 사용
- MatchService에서 `code_url` 대신 `docker_image_url` 사용
- Executor에 이미지 URL 전달
- Executor가 K8s Pod로 Agent 실행

### 4. 이미지 캐싱 및 최적화
- Kaniko 캐시 활성화 (현재: `--cache=true --cache-ttl=24h`)
- 레이어 캐싱으로 빌드 시간 단축
- 이미지 크기 최적화

### 5. CI/CD 파이프라인
- GitHub Actions 통합
- 자동 테스트 및 배포
- Staging/Production 환경 분리

## 주요 성과

✅ **gRPC 통신**: HTTP → gRPC 전환으로 성능 향상 (저지연, 양방향 스트리밍 가능)  
✅ **K8s Native**: K8s Service DNS 활용, in-cluster 통신 최적화  
✅ **컨테이너화**: Agent 코드를 Docker 이미지로 빌드하여 격리된 실행 환경 제공  
✅ **확장성**: Kaniko Job으로 여러 Agent 동시 빌드 가능  
✅ **보안**: Registry Secret으로 이미지 안전하게 저장  

## 문의 및 이슈
- BuilderService 초기화 실패: K8s 환경 외부에서는 경고만 표시하고 계속 진행
- Kaniko Job 타임아웃: 기본 30초, 필요 시 조정 가능
- Registry 인증: Secret 올바르게 생성되었는지 확인 필요

---

**Phase 1 완료일**: 2025년 1월  
**다음 Phase**: Phase 2 - 빌드 모니터링 및 Match 실행 통합
