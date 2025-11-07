# Pong Agent 제출 및 실행 - 간단 요약

## ✅ **네, 완벽하게 작동합니다!**

Pong 환경에서 학습시킨 Agent를 제출하면:
1. ✅ Docker 이미지로 자동 빌드
2. ✅ Executor가 K8s에서 두 Agent를 Pong 환경에 실행
3. ✅ 실제 게임 진행 및 승패 결정
4. ✅ 결과 렌더링 및 Replay 저장
5. ✅ ELO 점수 업데이트

---

## 🚀 빠른 시작

### 제출 방법 선택

**현재 시스템은 두 가지 방식을 모두 지원합니다:**

#### 방법 A: 파일 직접 업로드 (권장) ⭐
- Frontend에서 파일 선택하여 바로 업로드
- GitHub 없이도 사용 가능
- **간단하고 빠름!**

#### 방법 B: GitHub Repository URL
- GitHub에 코드 업로드 후 URL 제공
- 버전 관리 가능
- 협업에 유리

---

### 방법 A: 파일 직접 업로드 ⭐

#### 1. Agent 코드 작성 (agent.py)

```python
def get_action(observation):
    """
    observation: [ball_x, ball_y, ball_vx, ball_vy, paddle_y, opponent_y]
    return: 0 (STAY), 1 (UP), 2 (DOWN)
    """
    ball_y = observation[1]
    paddle_y = observation[4]
    
    if ball_y > paddle_y:
        return 2  # DOWN
    elif ball_y < paddle_y:
        return 1  # UP
    return 0  # STAY
```

#### 2. Dockerfile 작성

```dockerfile
FROM python:3.11-slim
RUN pip install rl-arena-env numpy
COPY agent.py /app/agent.py
WORKDIR /app
CMD ["python"]
```

#### 3. Web UI에서 제출

```
1. RL Arena 웹사이트 접속
2. Competition > Pong 선택
3. "Submit Agent" 버튼 클릭
4. 파일 선택:
   - agent.py 업로드
   - Dockerfile 업로드
   - requirements.txt (선택사항)
5. Submit 버튼 클릭
```

#### 또는 API로 제출

```bash
# 파일 업로드 방식
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Authorization: Bearer $TOKEN" \
  -F "agentId=your-agent-id" \
  -F "file=@agent.py"
```

---

### 방법 B: GitHub Repository URL

#### 1. Agent 코드 작성 (위와 동일)

#### 2. GitHub Repository 구성

```
my-pong-agent/
├── agent.py          # Agent 코드
├── Dockerfile        # Docker 빌드 설정
├── requirements.txt  # Python 의존성 (선택)
└── README.md         # 설명 (선택)
```

#### 3. GitHub에 업로드

```bash
git init
git add agent.py Dockerfile
git commit -m "Add Pong agent"
git push origin main
```

#### 4. Backend에 제출

```bash
# Web UI에서
Competition > Pong > Submit Agent
→ GitHub URL 입력: https://github.com/username/my-pong-agent

# 또는 API
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "agentId": "your-agent-id",
    "codeURL": "https://github.com/username/my-pong-agent"
  }'
```

### 5. 빌드 완료 대기

```bash
# 상태 확인
GET /api/v1/submissions/{id}/build-status
→ pending → building → active ✅

# 실패 시 재시도
POST /api/v1/submissions/{id}/rebuild
```

### 6. Match 생성

```bash
POST /api/v1/matches
{
  "agent1Id": "your-agent",
  "agent2Id": "opponent-agent",
  "environmentId": "pong"
}
```

### 7. 결과 확인

```bash
GET /api/v1/matches/{id}
→ winner, scores, replayUrl

GET /api/v1/leaderboard?environmentId=pong
→ ELO 순위
```

---

## 🎮 실행 과정

**두 가지 방식 모두 동일한 과정:**

```
Agent 제출 (파일 업로드 또는 GitHub URL)
  ↓
Backend에 파일 저장 / GitHub에서 클론
  ↓
Kaniko 빌드 (자동)
  ↓
Docker Image Push
  ↓
Match 생성 (수동/자동)
  ↓
Executor → K8s Job
  ↓
Orchestrator가 Pong 환경 생성
  ↓
Agent1 vs Agent2 실제 대결
  ↓
Replay 렌더링 및 저장
  ↓
결과 Backend에 반환
  ↓
ELO 점수 업데이트
  ↓
Leaderboard 갱신
```

---

## 📋 필수 파일

```
my-pong-agent/
├── agent.py          ✅ 필수 - Agent 코드
├── Dockerfile        ✅ 필수 - Docker 빌드 설정
├── requirements.txt  ⭐ 권장 - Python 의존성
└── README.md         📝 선택 - 설명
```

---

## 🔍 Agent 인터페이스

### 방법 1: 함수 형태 (간단)
```python
def get_action(observation):
    # 로직
    return action  # 0, 1, 2
```

### 방법 2: 클래스 형태
```python
class Agent:
    def __init__(self):
        pass
    
    def get_action(self, observation):
        return action
```

### 방법 3: rl-arena-env Agent 클래스
```python
from rl_arena.core.agent import Agent

class MyAgent(Agent):
    def act(self, observation):
        return action
```

**모두 작동합니다!** Orchestrator가 자동으로 인식합니다.

---

## 🎯 Pong 환경 스펙

**Observation (6차원):**
- `[0]` ball_x: 공 X 좌표 (-1 ~ 1)
- `[1]` ball_y: 공 Y 좌표 (-1 ~ 1)
- `[2]` ball_vx: 공 X 속도
- `[3]` ball_vy: 공 Y 속도
- `[4]` paddle_y: 내 패들 Y 좌표 (-1 ~ 1)
- `[5]` opponent_y: 상대 패들 Y 좌표 (-1 ~ 1)

**Action (3가지):**
- `0`: STAY - 정지
- `1`: UP - 위로 이동
- `2`: DOWN - 아래로 이동

**Reward:**
- `+1`: 상대가 공을 놓침
- `-1`: 내가 공을 놓침
- `0`: 그 외

---

## 💡 Agent 학습 예시

### DQN으로 학습
```python
import rl_arena

# 학습
model = rl_arena.train_dqn(
    env_name="pong",
    total_timesteps=100000,
    verbose=1
)
model.save("pong_agent.zip")

# Submission 생성
agent = rl_arena.create_agent(model)
rl_arena.create_submission(
    agent=agent,
    output_path="agent.py",
    agent_name="MyDQNAgent"
)
```

---

## 🚨 자주 발생하는 문제

### 빌드 실패
❌ **Dockerfile not found**
→ Dockerfile 파일명 확인 (대소문자 구분)

❌ **agent.py not found**
→ COPY 경로 확인

❌ **Module import error**
→ requirements.txt에 의존성 추가

### Match 실행 실패
❌ **No get_action method**
→ 함수/메서드 이름 확인

❌ **Invalid action**
→ 반환값이 0, 1, 2 중 하나인지 확인

❌ **Timeout**
→ Agent 코드 최적화 필요

---

## 📊 현재 시스템 상태

### ✅ 완벽하게 작동
- Agent 제출 및 자동 빌드
- 빌드 모니터링 (10초 폴링)
- 빌드 재시도 (최대 3회)
- Match 실행 (Pong 환경)
- 결과 저장 및 ELO 업데이트
- Leaderboard 조회

### ⚠️ 수동 작업 필요
- Match 생성 (자동 매칭 미구현)

### 🔜 개선 예정
- Replay 다운로드 API
- 실시간 알림 (WebSocket)
- Watch API (폴링 → 실시간)

---

## 📚 더 자세한 정보

- **전체 가이드:** `/docs/AGENT_SUBMISSION_GUIDE.md`
- **시스템 상태:** `/SYSTEM_STATUS.md`
- **API 문서:** `/API_DOCUMENTATION.md`
- **Phase 2 완료:** `/docs/PHASE_2_COMPLETE.md`
- **rl-arena-env:** `/rl-arena-env/README.md`

---

## 🎉 결론

**현재 시스템으로 완전히 작동합니다!**

Pong 환경에 맞게 학습시킨 Agent를 제출하면:
1. ✅ 자동으로 Docker 이미지 빌드
2. ✅ Executor가 K8s에서 실제 대결 진행
3. ✅ Replay 렌더링 및 저장
4. ✅ ELO 점수 자동 계산
5. ✅ Leaderboard 업데이트

**핵심 흐름 100% 완료!** 🚀

유일한 수동 작업은 Match 생성뿐입니다.
(자동 매칭은 Phase 4에서 구현 예정)
