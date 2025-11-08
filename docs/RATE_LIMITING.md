# Match Rate Limiting System

RL Arena는 Kaggle 경진대회 시스템을 참고하여 매치 횟수 제한 시스템을 구현했습니다.

## 📊 Rate Limit 정책

### 1. **일일 매치 제한** (Daily Match Limit)
- **기본값**: 에이전트당 하루 100 매치
- **목적**: 시스템 리소스 보호 및 공정한 경쟁 환경 조성
- **동작**: 자정(00:00)에 자동으로 카운터 리셋

### 2. **매치 쿨다운** (Match Cooldown)
- **기본값**: 매치 간 최소 5분 간격
- **목적**: 빠른 반복 제출 방지 및 시스템 안정성 확보
- **동작**: 마지막 매치 완료 시각부터 쿨다운 시간 측정

### 3. **동시 매치 제한** (Concurrent Match Limit)
- **기본값**: 에이전트당 동시에 1개 매치만 진행
- **목적**: 리소스 분산 및 안정적인 매치 실행
- **동작**: SQL 쿼리에서 `pending` 또는 `running` 상태의 매치 확인

## 🔧 설정 변경 방법

### 코드에서 설정 변경

`internal/models/agent_match_stats.go`의 `DefaultMatchRateLimitConfig` 함수 수정:

```go
func DefaultMatchRateLimitConfig() MatchRateLimitConfig {
    return MatchRateLimitConfig{
        DailyMatchLimit:    100,              // 하루 최대 매치 수
        MatchCooldown:      5 * time.Minute,  // 매치 간 대기 시간
        MaxConcurrentMatch: 1,                // 동시 매치 수
    }
}
```

### 환경별 권장 설정

#### 개발 환경 (빠른 테스트)
```go
DailyMatchLimit:    1000,
MatchCooldown:      1 * time.Minute,
```

#### 프로덕션 환경 (안정성 우선)
```go
DailyMatchLimit:    50,
MatchCooldown:      10 * time.Minute,
```

#### 경진대회 환경 (Kaggle 스타일)
```go
DailyMatchLimit:    5,                // 하루 5회 제한
MatchCooldown:      1 * time.Hour,    // 1시간 간격
```

## 📈 모니터링

### 에이전트별 매치 통계 조회

```sql
SELECT 
    a.name,
    ams.matches_today,
    ams.total_matches,
    ams.last_match_at,
    ams.daily_reset_at
FROM agents a
LEFT JOIN agent_match_stats ams ON a.id = ams.agent_id;
```

### Rate Limit 상태 확인

```sql
-- 스크립트 실행
\i scripts/test_rate_limits.sql
```

## 🧪 테스트

### 1. 쿨다운 테스트
```sql
-- 특정 에이전트의 마지막 매치 시각을 2분 전으로 설정
UPDATE agent_match_stats 
SET last_match_at = NOW() - INTERVAL '2 minutes' 
WHERE agent_id = '<AGENT_ID>';

-- 30초 후 다음 매칭 사이클에서 해당 에이전트는 제외됨 (5분 쿨다운)
```

### 2. 일일 제한 테스트
```sql
-- 특정 에이전트의 오늘 매치 수를 98로 설정
UPDATE agent_match_stats 
SET matches_today = 98 
WHERE agent_id = '<AGENT_ID>';

-- 2번 매치 후 해당 에이전트는 자동으로 매칭 제외됨
```

### 3. 통합 테스트
```bash
# 1. 데이터베이스 리셋
docker exec rl-arena-backend-db-1 psql -U postgres -d rl_arena -f /scripts/reset_database.sql

# 2. 백엔드 시작
go run cmd/server/main.go

# 3. 에이전트 생성 및 제출

# 4. 매칭 로그 확인
# "Starting matchmaking" 로그에서 cooldown_minutes, daily_limit 확인
```

## 📝 로그 메시지

### 정상 매칭
```
INFO Starting matchmaking env=pong waiting=4 cooldown_minutes=5 daily_limit=100
```

### Rate Limit 적용
```
DEBUG Not enough agents for matching env=pong count=2 cooldown_minutes=5 daily_limit=100
```

## 🔄 자동 리셋

### 일일 카운터 리셋
- **시각**: 매일 자정 (00:00)
- **방식**: SQL의 `CASE` 문으로 자동 처리
- **필드**: `daily_reset_at` 타임스탬프로 관리

```sql
-- IncrementMatchCount 함수의 자동 리셋 로직
matches_today = CASE 
    WHEN daily_reset_at <= NOW() THEN 1
    ELSE matches_today + 1
END
```

## 🎯 Kaggle과의 비교

| 기능 | Kaggle | RL Arena |
|------|--------|----------|
| 일일 제출 제한 | 5-10회 | 100회 (조정 가능) |
| 제출 간격 | 1시간 | 5분 (조정 가능) |
| 동시 평가 | 1개 | 1개 |
| 리셋 시각 | UTC 00:00 | 시스템 시간 00:00 |
| 쿨다운 표시 | ❌ | ✅ (로그) |
| 통계 추적 | ❌ | ✅ (DB) |

## 🚀 향후 개선 계획

1. **동적 Rate Limit**: ELO 등급별 차등 제한
2. **우선순위 시스템**: 오래 기다린 에이전트 우선 매칭
3. **Rate Limit API**: 프론트엔드에서 남은 횟수 표시
4. **알림 시스템**: 쿨다운 종료 또는 리셋 시각 알림
5. **관리자 도구**: Rate Limit 실시간 조정 기능

## 📚 참고 문서

- Kaggle Competition Rules: https://www.kaggle.com/competitions
- ELO Rating System: https://en.wikipedia.org/wiki/Elo_rating_system
- PostgreSQL Interval Types: https://www.postgresql.org/docs/current/datatype-datetime.html
