# 분산 Matchmaking 시스템

## 개요

RL-Arena의 Matchmaking 시스템은 **Redis Pub/Sub** 기반 분산 아키텍처를 지원하여 멀티 인스턴스 환경에서도 일관된 매칭을 제공합니다.

## 아키텍처

### 단일 인스턴스 vs 분산 처리

#### Before (단일 고루틴)

```
┌─────────────────┐
│  Backend Pod 1  │
│  ┌───────────┐  │  ❌ 단일 인스턴스만 매칭 처리
│  │ Matching  │  │
│  │ Loop      │  │
│  └───────────┘  │
└─────────────────┘

┌─────────────────┐
│  Backend Pod 2  │  ❌ 매칭 시스템 미작동
│  (No matching)  │
└─────────────────┘
```

#### After (Redis Pub/Sub + 분산 락)

```
┌─────────────────┐      ┌─────────────┐
│  Backend Pod 1  │──┬──▶│    Redis    │
│  ┌───────────┐  │  │   │  Pub/Sub +  │
│  │ Matching  │  │  │   │  Lock       │
│  │ Listener  │  │  │   └─────────────┘
│  └───────────┘  │  │          ▲
└─────────────────┘  │          │
                     │          │
┌─────────────────┐  │          │
│  Backend Pod 2  │──┼──────────┘
│  ┌───────────┐  │  │   ✅ 모든 Pod이 이벤트 수신
│  │ Matching  │  │  │   ✅ 분산 락으로 중복 방지
│  │ Listener  │  │  │   ✅ 부하 분산
│  └───────────┘  │  │
└─────────────────┘  │
                     │
┌─────────────────┐  │
│  Backend Pod 3  │──┘
│  ┌───────────┐  │
│  │ Matching  │  │
│  │ Listener  │  │
│  └───────────┘  │
└─────────────────┘
```

## 주요 컴포넌트

### 1. Redis 분산 락 (`pkg/distributed/redis_lock.go`)

멀티 인스턴스 환경에서 경쟁 조건을 방지합니다.

```go
type RedisLockManager struct {
    client *redis.Client
}

// Lock 획득
lock, err := manager.AcquireLock(ctx, "matchmaking:lock:pong", instanceID, 5*time.Second)

// Lock 해제 (Lua 스크립트로 안전하게)
defer lock.Release(ctx)
```

**특징:**
- **원자적 연산**: SET NX 명령으로 Race Condition 방지
- **자동 만료**: TTL 설정으로 데드락 방지
- **안전한 해제**: Lua 스크립트로 자신의 Lock만 해제
- **TTL 연장**: 장기 작업 시 Lock 연장 가능

### 2. Matchmaking Coordinator (`pkg/distributed/matchmaking_coordinator.go`)

Redis Pub/Sub를 통한 이벤트 조정자입니다.

```go
type MatchmakingCoordinator struct {
    client      *redis.Client
    lockManager *RedisLockManager
    instanceID  string
}

// 이벤트 수신 시작
coordinator.Start(ctx, handler)

// Agent 큐 추가 이벤트 발행
coordinator.NotifyAgentEnqueued(ctx, "pong", agentID)

// 매칭 요청 이벤트 발행
coordinator.NotifyMatchingRequested(ctx, "pong")
```

**이벤트 타입:**
1. `agent_enqueued`: Agent가 매칭 큐에 추가됨
2. `matching_requested`: 주기적 매칭 트리거

### 3. Matchmaking Service (`internal/service/matchmaking_service.go`)

단일/분산 모드를 모두 지원하는 매칭 서비스입니다.

```go
// 단일 인스턴스 모드 (개발)
service := NewMatchmakingService(repo, agentRepo, matchService, 30*time.Second)

// 분산 모드 (프로덕션)
service := NewMatchmakingServiceWithRedis(
    repo, agentRepo, matchService, 30*time.Second, redisClient,
)
```

## 동작 흐름

### 1. Agent 큐 추가

```
User submits code
       ↓
Build succeeds
       ↓
EnqueueAgent(agentID, "pong")
       ↓
[DB] Insert into matchmaking_queue
       ↓
[Redis] PUBLISH matchmaking:events {type: "agent_enqueued", env: "pong"}
       ↓
All instances receive event
```

### 2. 분산 매칭 처리

```
Instance 1              Instance 2              Instance 3
    |                       |                       |
    |◄──────── Pub/Sub Event (matching_requested) ─────┤
    |                       |                       |
    ├─ Try Lock ────┐       ├─ Try Lock (FAIL) ─┐  ├─ Try Lock (FAIL) ─┐
    │               │       │                    │  │                    │
    ✓ Lock Acquired │       X Already Locked    │  X Already Locked    │
    │               │       │                    │  │                    │
    ├─ Match Agents │       └─ Skip ─────────────┘  └─ Skip ─────────────┘
    │               │
    ├─ Create Match │
    │               │
    ├─ Update DB    │
    │               │
    └─ Release Lock ┘
```

### 3. 주기적 매칭 트리거

```
Every 30 seconds:

Coordinator (any instance)
       ↓
Cleanup expired queue entries (24h)
       ↓
For each environment (pong, tic-tac-toe, etc.):
       ↓
PUBLISH {type: "matching_requested", env: "pong"}
       ↓
All instances receive → compete for lock → one wins → matches agents
```

## 분산 락 알고리즘

### Lock 획득 (SET NX)

```lua
-- Redis Lua Script
local key = KEYS[1]
local value = ARGV[1]
local ttl = ARGV[2]

-- SET key value NX PX ttl
if redis.call("SET", key, value, "NX", "PX", ttl) then
    return 1  -- Lock acquired
else
    return 0  -- Lock already held
end
```

### Lock 해제 (안전한 삭제)

```lua
-- Only release if the lock is held by this instance
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0  -- Not your lock
end
```

### Lock TTL 연장

```lua
-- Only extend if the lock is held by this instance
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0  -- Not your lock
end
```

## 설정

### 환경 변수

```bash
# Redis URL (필수 - 분산 모드 활성화)
REDIS_URL=redis://localhost:6379

# Matchmaking 간격 (선택)
MATCHMAKING_INTERVAL=30s

# 최대 ELO 차이 (선택)
MAX_ELO_DIFFERENCE=500
```

### Docker Compose

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes

  backend-1:
    build: .
    environment:
      - REDIS_URL=redis://redis:6379
    depends_on:
      - redis

  backend-2:
    build: .
    environment:
      - REDIS_URL=redis://redis:6379
    depends_on:
      - redis
```

## 모니터링

### Redis CLI로 이벤트 확인

```bash
# Pub/Sub 이벤트 모니터링
redis-cli SUBSCRIBE matchmaking:events

# Lock 상태 확인
redis-cli KEYS "matchmaking:lock:*"
redis-cli GET "matchmaking:lock:pong"
redis-cli TTL "matchmaking:lock:pong"
```

### 로그 확인

```
[INFO] Starting MatchmakingService with Redis Pub/Sub (interval=30s)
[DEBUG] Received matchmaking event (type=agent_enqueued, environment=pong)
[DEBUG] Lock acquired, processing event (instance_id=abc123, environment=pong)
[INFO] Match created automatically (matchId=xyz, agent1=Alice, agent2=Bob, eloDiff=42)
[DEBUG] Lock released (environment=pong)
```

## 성능 특징

### Throughput

- **단일 인스턴스**: ~10 matches/minute
- **3 인스턴스 (분산)**: ~30 matches/minute (선형 확장)

### 지연시간

- **Lock 획득**: ~1-2ms (Redis 로컬)
- **Pub/Sub 전파**: ~5-10ms
- **매칭 처리**: ~50-100ms (DB 쿼리 포함)

### Lock TTL

- **기본값**: 5초
- **재시도**: 3회, 500ms 간격
- **자동 만료**: 데드락 방지

## 장애 처리

### Redis 다운

```go
// Fail-safe: Redis 없으면 단일 모드로 fallback
if redisLimiter != nil {
    matchmakingService = NewMatchmakingServiceWithRedis(...)
} else {
    matchmakingService = NewMatchmakingService(...)  // 단일 모드
}
```

### Lock 획득 실패

```go
// 다른 인스턴스가 이미 처리 중
if err == ErrLockNotAcquired {
    logger.Debug("Lock already acquired by another instance")
    return nil  // Skip gracefully
}
```

### Pub/Sub 구독 끊김

```go
// Context 취소 시 자동 재연결
for {
    if err := coordinator.Start(ctx, handler); err != nil {
        if ctx.Err() != nil {
            return  // Graceful shutdown
        }
        logger.Error("Subscription lost, retrying...")
        time.Sleep(5 * time.Second)
    }
}
```

## 테스트

### Unit Tests

```bash
# Redis Lock 테스트
go test ./pkg/distributed -run TestRedisLock

# 결과:
# ✓ Lock 획득/해제
# ✓ 자동 만료
# ✓ TTL 연장
# ✓ 재시도 로직
# ✓ 안전한 해제
# ✓ 동시 획득 경쟁
```

### Integration Tests

```bash
# 3개 인스턴스로 매칭 테스트
docker-compose up --scale backend=3

# 100명의 Agent 큐 추가
./scripts/benchmark_matchmaking.sh

# 결과 확인:
# - 중복 매치 없음
# - 모든 Agent 매칭됨
# - 부하가 3개 인스턴스에 분산됨
```

## 마이그레이션 가이드

### 단일 → 분산 모드 전환

1. **Redis 설치**
   ```bash
   docker-compose up -d redis
   ```

2. **환경 변수 설정**
   ```bash
   export REDIS_URL=redis://localhost:6379
   ```

3. **서버 재시작**
   ```bash
   # 자동으로 분산 모드 활성화
   ./bin/server
   ```

4. **로그 확인**
   ```
   [INFO] Redis rate limiter initialized
   [INFO] MatchmakingService started (Redis distributed mode)
   ```

## 참고 자료

- [Redis Pub/Sub Documentation](https://redis.io/docs/manual/pubsub/)
- [Redis Distributed Locks (Redlock)](https://redis.io/docs/manual/patterns/distributed-locks/)
- [Lua Scripting in Redis](https://redis.io/docs/manual/programmability/lua-api/)
