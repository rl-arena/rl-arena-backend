#!/usr/bin/env python3
"""Mock Executor for testing"""

from flask import Flask, request, jsonify
import random
import time

app = Flask(__name__)

@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "ok"})

@app.route('/execute', methods=['POST'])
def execute():
    data = request.json
    
    match_id = data['matchId']
    agent1 = data['agent1']
    agent2 = data['agent2']
    
    print(f"\n=== Executing match: {match_id} ===")
    print(f"  Agent1: {agent1['name']} ({agent1['id']})")
    print(f"  Agent2: {agent2['name']} ({agent2['id']})")
    
    # 게임 실행 시뮬레이션 (1-3초)
    time.sleep(random.uniform(1, 3))
    
    # 랜덤 결과 생성
    winner_choice = random.choice([agent1['id'], agent2['id'], None])
    
    if winner_choice == agent1['id']:
        agent1_score = 1.0
        agent2_score = 0.0
        winner_name = agent1['name']
    elif winner_choice == agent2['id']:
        agent1_score = 0.0
        agent2_score = 1.0
        winner_name = agent2['name']
    else:
        agent1_score = 0.5
        agent2_score = 0.5
        winner_name = "Draw"
    
    result = {
        "matchId": match_id,
        "status": "success",
        "winnerId": winner_choice,
        "agent1Score": agent1_score,
        "agent2Score": agent2_score,
        "replayUrl": f"https://replays.example.com/{match_id}.json",
        "duration": random.randint(1000, 5000)
    }
    
    print(f"  Result: {winner_name} wins!")
    print(f"  Scores: {agent1_score} - {agent2_score}")
    
    return jsonify(result)

if __name__ == '__main__':
    print("🚀 Mock Executor running on http://localhost:8081")
    app.run(port=8081, debug=True)
