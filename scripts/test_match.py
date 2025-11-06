#!/usr/bin/env python3
import requests
import json

BASE_URL = "http://localhost:8080"

# 1. 회원가입
print("1. 회원가입...")
resp = requests.post(f"{BASE_URL}/api/v1/auth/register", json={
    "username": "player1",
    "email": "player1@example.com",
    "password": "password123",
    "fullName": "Player One"
})

print(f"Status: {resp.status_code}")
print(f"Response: {resp.text}\n")

if resp.status_code != 201:
    print("❌ 회원가입 실패!")
    print("이미 가입된 사용자일 수 있습니다. 로그인 시도...")

    # 로그인 시도
    resp = requests.post(f"{BASE_URL}/api/v1/auth/login", json={
        "email": "player1@example.com",
        "password": "password123"
    })

    print(f"Login Status: {resp.status_code}")
    print(f"Login Response: {resp.text}\n")

    if resp.status_code != 200:
        print("❌ 로그인도 실패! 다른 이메일로 시도하세요.")
        exit(1)

data = resp.json()
TOKEN = data['token']
print(f"✅ Token: {TOKEN[:50]}...")

# 2. 첫 번째 에이전트 생성
print("\n2. Agent1 생성...")
resp = requests.post(f"{BASE_URL}/api/v1/agents",
    headers={"Authorization": f"Bearer {TOKEN}"},
    json={
        "name": "Smart Bot",
        "description": "A smart agent",
        "environmentId": "tic-tac-toe"
    })

print(f"Status: {resp.status_code}")
print(f"Response: {resp.text}\n")

if resp.status_code != 201:
    print("❌ Agent1 생성 실패!")
    exit(1)

AGENT1_ID = resp.json()['agent']['id']
print(f"✅ Agent1 ID: {AGENT1_ID}")

# 3. 두 번째 에이전트 생성
print("\n3. Agent2 생성...")
resp = requests.post(f"{BASE_URL}/api/v1/agents",
    headers={"Authorization": f"Bearer {TOKEN}"},
    json={
        "name": "Random Bot",
        "description": "A random agent",
        "environmentId": "tic-tac-toe"
    })

print(f"Status: {resp.status_code}")

if resp.status_code != 201:
    print("❌ Agent2 생성 실패!")
    exit(1)

AGENT2_ID = resp.json()['agent']['id']
print(f"✅ Agent2 ID: {AGENT2_ID}")

# 4. Agent1 코드 제출
print("\n4. Agent1 코드 제출...")
with open('agent1.py', 'w') as f:
    f.write("def make_move(board):\n    return (0, 0)")

with open('agent1.py', 'rb') as f:
    resp = requests.post(f"{BASE_URL}/api/v1/submissions",
        headers={"Authorization": f"Bearer {TOKEN}"},
        data={"agentId": AGENT1_ID},
        files={"file": f})

print(f"Status: {resp.status_code}")

if resp.status_code != 201:
    print(f"❌ Submission1 실패! {resp.text}")
    exit(1)

SUB1_ID = resp.json()['submission']['id']
print(f"✅ Submission1 ID: {SUB1_ID}")

# 5. Agent1 활성화
print("\n5. Agent1 활성화...")
resp = requests.put(f"{BASE_URL}/api/v1/submissions/{SUB1_ID}/activate",
    headers={"Authorization": f"Bearer {TOKEN}"})

if resp.status_code != 200:
    print(f"❌ 활성화 실패! {resp.text}")
    exit(1)

print("✅ Agent1 활성화됨")

# 6. Agent2 코드 제출
print("\n6. Agent2 코드 제출...")
with open('agent2.py', 'w') as f:
    f.write("def make_move(board):\n    return (1, 1)")

with open('agent2.py', 'rb') as f:
    resp = requests.post(f"{BASE_URL}/api/v1/submissions",
        headers={"Authorization": f"Bearer {TOKEN}"},
        data={"agentId": AGENT2_ID},
        files={"file": f})

print(f"Status: {resp.status_code}")

if resp.status_code != 201:
    print(f"❌ Submission2 실패! {resp.text}")
    exit(1)

SUB2_ID = resp.json()['submission']['id']
print(f"✅ Submission2 ID: {SUB2_ID}")

# 7. Agent2 활성화
print("\n7. Agent2 활성화...")
resp = requests.put(f"{BASE_URL}/api/v1/submissions/{SUB2_ID}/activate",
    headers={"Authorization": f"Bearer {TOKEN}"})

if resp.status_code != 200:
    print(f"❌ 활성화 실패! {resp.text}")
    exit(1)

print("✅ Agent2 활성화됨")

# 8. 매치 생성!
print("\n8. 매치 생성 및 실행...")
resp = requests.post(f"{BASE_URL}/api/v1/matches",
    headers={"Authorization": f"Bearer {TOKEN}"},
    json={
        "agent1Id": AGENT1_ID,
        "agent2Id": AGENT2_ID
    })

print(f"Status: {resp.status_code}")
print(f"Response: {resp.text}\n")

if resp.status_code != 201:
    print("❌ 매치 생성 실패!")
    exit(1)

result = resp.json()
print("\n🎮 Match Result:")
print(json.dumps(result, indent=2))

# 9. 리더보드 확인
print("\n9. 리더보드:")
resp = requests.get(f"{BASE_URL}/api/v1/leaderboard")
print(json.dumps(resp.json(), indent=2))

print("\n✅ 모든 테스트 완료!")
print(f"\n저장된 정보:")
print(f"TOKEN={TOKEN}")
print(f"AGENT1_ID={AGENT1_ID}")
print(f"AGENT2_ID={AGENT2_ID}")