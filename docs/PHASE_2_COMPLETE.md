# Phase 2 완료 - 빌드 모니터링 및 Match 실행 통합

## 개요
Phase 2에서는 Phase 1에서 구축한 Kaniko 빌드 파이프라인을 활성화하고, 빌드 상태를 자동으로 모니터링하며, Match 실행 시 Docker 이미지를 사용하도록 통합했습니다.

## 완료된 작업

### 1. 빌드 자동 시작 활성화 ✅

**파일:** `internal/service/service.go`

- **변경사항:**
  - CreateFromURL에서 TODO 주석 제거
  - BuilderService를 비동기로 호출하여 Docker 이미지 빌드 시작
  - zap Logger 추가하여 빌드 진행 상황 로깅
  - 빌드 실패 시 자동으로 상태 업데이트

- **로직 흐름:**
  ```
  Agent 제출 (GitHub URL)
    ↓
  Submission 생성 (status: pending)
    ↓
  BuilderService.BuildAgentImage() 비동기 호출
    ↓
  Submission 상태 → 'building'
    ↓
  Kaniko Job 생성
    ↓
  BuildMonitor가 상태 추적 시작
  ```

- **에러 핸들링:**
  - BuilderService 초기화 실패 시 경고 로그 출력 후 계속 진행
  - 빌드 실패 시 'build_failed' 상태로 업데이트 및 에러 메시지 저장

### 2. 빌드 상태 모니터링 서비스 구현 ✅

**파일:** `internal/service/build_monitor.go` (237 lines)

- **BuildMonitor 구조:**
  ```go
  type BuildMonitor struct {
      builderService *BuilderService
      submissionRepo *repository.SubmissionRepository
      logger         *zap.Logger
      checkInterval  time.Duration  // 기본 10초
      stopChan       chan struct{}
      wg             sync.WaitGroup
  }
  ```

- **주요 기능:**
  1. **Start()**: 백그라운드 모니터링 시작
  2. **checkAllBuilds()**: 10초마다 'building' 상태의 모든 Submission 조회
  3. **checkBuild()**: 개별 Submission의 K8s Job 상태 확인
  4. **handleBuildSuccess()**: 빌드 성공 시 처리
     - 상태를 'success'로 업데이트
     - 빌드 로그 가져와서 DB에 저장
  5. **handleBuildFailure()**: 빌드 실패 시 처리
     - 상태를 'build_failed'로 업데이트
     - 에러 메시지 및 로그 저장

- **Repository 추가:**
  - `FindByStatus(status)`: 특정 상태의 모든 Submission 조회

- **Router 통합:**
  ```go
  buildMonitor := service.NewBuildMonitor(builderService, submissionRepo, 10*time.Second)
  buildMonitor.Start()
  ```

### 3. Submission 상태 전이 로직 ✅

**파일:** `internal/models/submission.go`

- **새로운 상태 추가:**
  ```go
  const (
      SubmissionStatusPending     SubmissionStatus = "pending"
      SubmissionStatusBuilding    SubmissionStatus = "building"
      SubmissionStatusSuccess     SubmissionStatus = "success"      // 신규
      SubmissionStatusBuildFailed SubmissionStatus = "build_failed" // 신규
      SubmissionStatusActive      SubmissionStatus = "active"
      SubmissionStatusFailed      SubmissionStatus = "failed"
      SubmissionStatusInactive    SubmissionStatus = "inactive"
  )
  ```

- **상태 전이 다이어그램:**
  ```
  pending
     ↓
  building
     ↓ (성공)
  success → active (사용자가 활성화)
     ↓ (실패)
  build_failed
  ```

### 4. Match 실행 시 Docker 이미지 사용 ✅

**파일:** `internal/service/match_service.go`

- **getAgentSource() 메서드 추가:**
  ```go
  func (s *MatchService) getAgentSource(submission *models.Submission) (string, bool) {
      // 1. Docker 이미지 우선 사용 (빌드 성공한 경우)
      if submission.DockerImageURL != nil && *submission.DockerImageURL != "" {
          if submission.Status == models.SubmissionStatusSuccess ||
             submission.Status == models.SubmissionStatusActive {
              return *submission.DockerImageURL, true
          }
      }
      
      // 2. 빌드 중이면 대기 필요
      if submission.Status == models.SubmissionStatusBuilding {
          return "", false
      }
      
      // 3. Fallback: 코드 URL 사용 (backward compatibility)
      return submission.CodeURL, false
  }
  ```

- **CreateAndExecute() 수정:**
  - Submission 조회 후 getAgentSource() 호출
  - Docker 이미지 또는 코드 URL 결정
  - 빌드 중인 Submission은 Match 생성 거부
  - Executor에 적절한 소스 전달

- **로깅 개선:**
  ```go
  logger.Info("Match created",
      "agent1Source", agent1Source,
      "agent1IsDocker", agent1IsDocker,
      "agent2Source", agent2Source,
      "agent2IsDocker", agent2IsDocker,
  )
  ```

### 5. 빌드 관련 API 엔드포인트 추가 ✅

**파일:** `internal/api/handlers/submission.go`

#### 새로운 엔드포인트:

1. **GET /api/v1/submissions/:id/build-status**
   - 빌드 상태 조회
   - 응답:
     ```json
     {
       "submissionId": "sub-123",
       "status": "building",
       "buildJobName": "build-sub-123",
       "buildPodName": "build-sub-123-abc123",
       "dockerImageUrl": "docker.io/myregistry/agent-123:sub-123-v1",
       "createdAt": "2025-01-07T10:00:00Z",
       "updatedAt": "2025-01-07T10:05:00Z"
     }
     ```

2. **GET /api/v1/submissions/:id/build-logs**
   - 빌드 로그 조회
   - 응답:
     ```json
     {
       "submissionId": "sub-123",
       "status": "success",
       "buildLog": "Step 1/5 : FROM python:3.9...",
       "updatedAt": "2025-01-07T10:10:00Z"
     }
     ```
   - 로그가 없으면 404 반환

**Router 업데이트:**
```go
submissions.GET("/:id/build-status", submissionHandler.GetBuildStatus)
submissions.GET("/:id/build-logs", submissionHandler.GetBuildLogs)
```

---

## 아키텍처 변경사항

### Before (Phase 1)
```
[Client] → [Backend] → [BuilderService]
                           ↓
                       [Kaniko Job] (수동, TODO 주석)
```

### After (Phase 2)
```
[Client] → POST /submissions (GitHub URL)
              ↓
        [SubmissionService]
              ↓ (비동기)
        [BuilderService] → [K8s Kaniko Job]
              ↓
        [BuildMonitor] (백그라운드, 10초마다)
              ↓ Job 상태 확인
        [K8s API]
              ↓ 상태 업데이트
        [SubmissionRepo] → [PostgreSQL]
              ↓
        [MatchService] → docker_image_url 사용
              ↓
        [Executor gRPC] → Match 실행
```

---

## 주요 파일 목록

### 새로 생성된 파일
1. `internal/service/build_monitor.go` - 빌드 모니터링 서비스 (237 lines)
2. `docs/PHASE_2_COMPLETE.md` - Phase 2 완료 문서

### 수정된 파일
1. `internal/service/service.go`
   - CreateFromURL: 비동기 빌드 시작 로직 추가
   - Logger 추가

2. `internal/service/match_service.go`
   - getAgentSource() 메서드 추가
   - CreateAndExecute(): Docker 이미지 우선 사용

3. `internal/models/submission.go`
   - SubmissionStatusSuccess 추가
   - SubmissionStatusBuildFailed 추가

4. `internal/repository/submission_repository.go`
   - FindByStatus() 메서드 추가

5. `internal/api/handlers/submission.go`
   - GetBuildStatus() 핸들러 추가
   - GetBuildLogs() 핸들러 추가

6. `internal/api/router.go`
   - BuildMonitor 초기화 및 시작
   - 빌드 관련 엔드포인트 라우팅 추가
   - time import 추가

---

## 빌드 검증

```bash
# 모든 변경사항 후 빌드 성공 확인
go build ./...  ✅

# 에러 없음, 모든 패키지 컴파일 성공
```

---

## API 테스트 시나리오

### 1. Agent 제출 및 빌드 시작
```bash
# 1. Agent 제출 (GitHub URL)
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "agentId": "agent-123",
    "codeUrl": "https://github.com/username/my-agent.git"
  }'

# 응답:
{
  "message": "Submission created successfully",
  "submission": {
    "id": "sub-abc123",
    "agentId": "agent-123",
    "version": 1,
    "status": "building",
    "codeUrl": "https://github.com/username/my-agent.git",
    "createdAt": "2025-01-07T10:00:00Z"
  }
}
```

### 2. 빌드 상태 확인
```bash
# 빌드 상태 조회 (즉시)
curl http://localhost:8080/api/v1/submissions/sub-abc123/build-status

# 응답 (빌드 중):
{
  "submissionId": "sub-abc123",
  "status": "building",
  "buildJobName": "build-sub-abc123",
  "buildPodName": "build-sub-abc123-xyz789",
  "createdAt": "2025-01-07T10:00:00Z",
  "updatedAt": "2025-01-07T10:00:05Z"
}

# 10초 후 (빌드 완료):
{
  "submissionId": "sub-abc123",
  "status": "success",
  "buildJobName": "build-sub-abc123",
  "dockerImageUrl": "docker.io/myregistry/agent-123:sub-abc123-v1",
  "createdAt": "2025-01-07T10:00:00Z",
  "updatedAt": "2025-01-07T10:02:30Z"
}
```

### 3. 빌드 로그 조회
```bash
# 빌드 로그 가져오기
curl http://localhost:8080/api/v1/submissions/sub-abc123/build-logs

# 응답:
{
  "submissionId": "sub-abc123",
  "status": "success",
  "buildLog": "Step 1/5 : FROM python:3.9-slim\n ---> abc123def456\nStep 2/5 : WORKDIR /app\n...",
  "updatedAt": "2025-01-07T10:02:30Z"
}
```

### 4. Match 생성 (Docker 이미지 사용)
```bash
# 빌드 완료된 Agent로 Match 생성
curl -X POST http://localhost:8080/api/v1/matches \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "agent1Id": "agent-123",
    "agent2Id": "agent-456"
  }'

# Backend 로그 확인:
INFO  Match created  matchId=match-xyz  agent1=MyAgent  agent1Source=docker.io/myregistry/agent-123:sub-abc123-v1  agent1IsDocker=true  agent2=OpponentAgent  agent2Source=docker.io/myregistry/agent-456:sub-def456-v2  agent2IsDocker=true
```

---

## 에러 케이스 처리

### 1. 빌드 중인 Agent로 Match 시도
```bash
curl -X POST http://localhost:8080/api/v1/matches ...

# 응답 (400 Bad Request):
{
  "error": "agent1 submission is not ready (status: building)"
}
```

### 2. 빌드 실패한 Submission
```bash
curl http://localhost:8080/api/v1/submissions/sub-failed123/build-status

# 응답:
{
  "submissionId": "sub-failed123",
  "status": "build_failed",
  "errorMessage": "Build failed: Error during kaniko build...",
  "updatedAt": "2025-01-07T10:05:00Z"
}
```

### 3. 로그가 없는 Submission
```bash
curl http://localhost:8080/api/v1/submissions/sub-pending123/build-logs

# 응답 (404 Not Found):
{
  "error": "Build logs not available",
  "message": "Logs may not be available yet or build has not started"
}
```

---

## 성능 및 모니터링

### BuildMonitor 성능
- **폴링 주기:** 10초
- **동시성:** goroutine 사용으로 논블로킹
- **리소스 사용:**
  - 'building' 상태 Submission이 0개일 때: 최소 리소스
  - 10개 빌드 동시 진행 시: K8s API 호출 10회/10초

### 로깅
- **정보 로그:**
  - 빌드 시작
  - 빌드 성공/실패
  - Match 생성 (소스 타입 포함)
  
- **에러 로그:**
  - BuildMonitor 초기화 실패
  - K8s API 호출 실패
  - 상태 업데이트 실패

---

## 향후 개선 사항 (Phase 3 후보)

### 1. 실시간 알림
- WebSocket을 통한 빌드 상태 실시간 업데이트
- 클라이언트가 폴링 없이 즉시 상태 변경 수신

### 2. 빌드 재시도
- `POST /submissions/:id/rebuild` 엔드포인트
- 빌드 실패 시 자동 재시도 (최대 3회)
- Exponential backoff

### 3. 빌드 캐싱 최적화
- Kaniko 캐시 효과 측정
- Layer 캐싱으로 빌드 시간 단축

### 4. K8s Watch API 사용
- Polling 대신 Watch API로 즉시 상태 변경 감지
- 리소스 사용 최소화

### 5. 빌드 우선순위
- 중요한 Agent 빌드 우선 처리
- 빌드 큐 관리

### 6. 이미지 스캔
- Trivy 통합하여 보안 취약점 자동 검사
- 취약점 발견 시 경고

---

## 배포 체크리스트

✅ **코드 변경사항:**
- [x] SubmissionService: 비동기 빌드 시작
- [x] BuildMonitor: 백그라운드 모니터링
- [x] MatchService: Docker 이미지 우선 사용
- [x] API 핸들러: 빌드 상태/로그 엔드포인트
- [x] Router: 새 엔드포인트 라우팅

✅ **빌드 검증:**
- [x] `go build ./...` 성공
- [x] 모든 패키지 컴파일 완료

✅ **문서화:**
- [x] PHASE_2_COMPLETE.md 작성
- [x] API 엔드포인트 문서화
- [x] 테스트 시나리오 작성

⏸️ **배포 준비 (사용자 작업 필요):**
- [ ] DB 마이그레이션 적용 (003_add_docker_image.sql)
- [ ] Container Registry Secret 생성
- [ ] ConfigMap 업데이트 (USE_K8S=true)
- [ ] Backend 재배포
- [ ] E2E 테스트 실행

---

## 주요 성과

✅ **자동화:** Agent 제출 시 자동으로 Docker 이미지 빌드 시작  
✅ **모니터링:** 10초마다 빌드 상태 자동 추적 및 업데이트  
✅ **통합:** Match 실행 시 Docker 이미지 자동 사용  
✅ **API:** 빌드 상태 및 로그 조회 엔드포인트 제공  
✅ **에러 처리:** 빌드 실패, 빌드 중인 Agent 등 모든 케이스 처리  

---

## Phase 2 vs Phase 1 비교

| 항목 | Phase 1 | Phase 2 |
|------|---------|---------|
| 빌드 시작 | TODO 주석 (수동) | 자동 (비동기) |
| 빌드 모니터링 | 없음 | BuildMonitor (10초 폴링) |
| 상태 관리 | pending, building | +success, +build_failed |
| Match 실행 | code_url 사용 | docker_image_url 우선 |
| API | 없음 | build-status, build-logs |
| 로깅 | 기본 | 상세 (zap logger) |

---

**Phase 2 완료일**: 2025년 1월 7일  
**다음 Phase**: Phase 3 - 실시간 알림, 빌드 재시도, 캐싱 최적화
