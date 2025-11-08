# RL Arena Backend - Kubernetes Deployment Guide

## 📋 배포 순서

이 가이드는 RL Arena Backend를 Kubernetes 클러스터에 배포하는 방법을 설명합니다.

## 🚀 Phase 0: Infrastructure Setup

### 1. 사전 요구사항

#### 로컬 Kubernetes 클러스터 (선택)
```bash
# Minikube 사용
minikube start --memory=4096 --cpus=2

# Kind 사용
kind create cluster --name rl-arena

# Docker Desktop의 Kubernetes 활성화
# Docker Desktop > Settings > Kubernetes > Enable Kubernetes
```

#### kubectl 설치 확인
```bash
kubectl version --client
```

### 2. Namespace 생성

```bash
cd rl-arena-backend/k8s
kubectl apply -f namespace.yaml
```

확인:
```bash
kubectl get namespaces | grep rl-arena
```

### 3. ConfigMap 및 Secret 설정

#### ConfigMap 적용 (환경 변수)
```bash
kubectl apply -f configmap.yaml
```

#### Secret 수정 및 적용
⚠️ **중요**: `secret.yaml` 파일에서 프로덕션 환경에 맞게 비밀번호를 변경하세요!

```bash
# secret.yaml 편집
# - JWT_SECRET 변경
# - POSTGRES_PASSWORD 변경
# - DATABASE_URL 업데이트

kubectl apply -f secret.yaml
```

#### Registry Credentials 생성 (Docker Hub/Private Registry)
```bash
# Docker Hub 사용 시
kubectl create secret docker-registry registry-credentials \
  --docker-server=docker.io \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  --docker-email=YOUR_EMAIL \
  --namespace=rl-arena

# Harbor/Private Registry 사용 시
kubectl create secret docker-registry registry-credentials \
  --docker-server=YOUR_REGISTRY_URL \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  --namespace=rl-arena
```

확인:
```bash
kubectl get configmap -n rl-arena
kubectl get secret -n rl-arena
```

### 4. Database 배포 (PostgreSQL StatefulSet)

```bash
kubectl apply -f postgres-statefulset.yaml
```

배포 상태 확인:
```bash
kubectl get statefulset -n rl-arena
kubectl get pods -n rl-arena | grep postgres

# PostgreSQL 로그 확인
kubectl logs -n rl-arena postgres-0 -f
```

PostgreSQL 준비 대기:
```bash
kubectl wait --for=condition=ready pod/postgres-0 -n rl-arena --timeout=300s
```

### 5. Redis 배포

```bash
kubectl apply -f redis-deployment.yaml
```

배포 상태 확인:
```bash
kubectl get deployment -n rl-arena | grep redis
kubectl get pods -n rl-arena | grep redis

# Redis 로그 확인
kubectl logs -n rl-arena -l app=redis -f
```

### 6. Backend Docker 이미지 빌드 및 푸시

#### Backend 이미지 빌드
```bash
cd rl-arena-backend

# Docker 이미지 빌드
docker build -t YOUR_REGISTRY/rl-arena-backend:latest .

# Registry에 푸시
docker push YOUR_REGISTRY/rl-arena-backend:latest
```

#### deployment.yaml 수정
`k8s/deployment.yaml` 파일에서 이미지 이름 변경:
```yaml
spec:
  containers:
  - name: backend
    image: YOUR_REGISTRY/rl-arena-backend:latest  # 여기를 수정
```

#### configmap.yaml 수정
`k8s/configmap.yaml` 파일에서 Registry 설정:
```yaml
data:
  CONTAINER_REGISTRY: "YOUR_REGISTRY"  # 여기를 수정
```

### 7. Backend 배포

```bash
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

배포 상태 확인:
```bash
kubectl get deployment -n rl-arena
kubectl get pods -n rl-arena | grep backend
kubectl get svc -n rl-arena

# Backend 로그 확인
kubectl logs -n rl-arena -l app=rl-arena-backend -f
```

### 8. Database 마이그레이션 실행

Backend Pod에 접속하여 마이그레이션 실행:
```bash
# Backend Pod 이름 확인
kubectl get pods -n rl-arena | grep backend

# Pod에 접속
kubectl exec -it -n rl-arena rl-arena-backend-xxxxx-xxxxx -- /bin/sh

# 마이그레이션 실행 (Pod 내부)
# Backend에 migrate 명령어가 있는 경우 실행
# 또는 psql로 직접 실행
```

또는 로컬에서 Port Forward로 마이그레이션:
```bash
# PostgreSQL Port Forward
kubectl port-forward -n rl-arena svc/postgres 5432:5432

# 로컬에서 마이그레이션 실행
psql "postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable" \
  -f migrations/001_initial_schema.sql

psql "postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable" \
  -f migrations/002_add_pong_environment.sql
```

### 9. Ingress 배포 (선택사항)

#### NGINX Ingress Controller 설치
```bash
# Minikube의 경우
minikube addons enable ingress

# Kind의 경우
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# 일반 클러스터의 경우
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml
```

#### Ingress 적용
```bash
kubectl apply -f ingress.yaml
```

#### /etc/hosts 파일 수정 (로컬 테스트용)
```bash
# Minikube의 경우
echo "$(minikube ip) rl-arena.local" | sudo tee -a /etc/hosts

# Docker Desktop의 경우
echo "127.0.0.1 rl-arena.local" | sudo tee -a /etc/hosts
```

### 10. 전체 시스템 상태 확인

```bash
# 모든 리소스 확인
kubectl get all -n rl-arena

# Pod 상태 확인
kubectl get pods -n rl-arena

# Service 확인
kubectl get svc -n rl-arena

# Endpoints 확인 (Service DNS 테스트)
kubectl get endpoints -n rl-arena

# Ingress 확인
kubectl get ingress -n rl-arena
```

## 🧪 테스트

### 1. Health Check
```bash
# Port Forward
kubectl port-forward -n rl-arena svc/rl-arena-backend 8080:8080

# Health Check 요청
curl http://localhost:8080/health
```

### 2. Ingress 테스트 (Ingress 배포 시)
```bash
curl http://rl-arena.local/api/health
```

### 3. Service DNS 테스트
```bash
# Backend Pod에서 Executor DNS 확인
kubectl exec -it -n rl-arena rl-arena-backend-xxxxx-xxxxx -- /bin/sh

# Pod 내부에서
nslookup rl-arena-executor.rl-arena.svc.cluster.local
ping rl-arena-executor.rl-arena.svc.cluster.local
```

## 🔧 Troubleshooting

### Pod이 CrashLoopBackOff 상태
```bash
# 로그 확인
kubectl logs -n rl-arena POD_NAME

# 이전 컨테이너 로그 확인
kubectl logs -n rl-arena POD_NAME --previous

# Pod 상세 정보
kubectl describe pod -n rl-arena POD_NAME
```

### Database 연결 실패
```bash
# PostgreSQL Pod 상태 확인
kubectl get pods -n rl-arena | grep postgres

# PostgreSQL 로그
kubectl logs -n rl-arena postgres-0

# Service Endpoints 확인
kubectl get endpoints -n rl-arena postgres

# Database 연결 테스트 (Backend Pod에서)
kubectl exec -it -n rl-arena rl-arena-backend-xxxxx-xxxxx -- /bin/sh
# apk add postgresql-client
# psql "postgres://postgres:password@postgres.rl-arena.svc.cluster.local:5432/rl_arena?sslmode=disable"
```

### Executor와 통신 실패
```bash
# Executor Service 확인
kubectl get svc -n rl-arena rl-arena-executor

# Executor Pod 확인
kubectl get pods -n rl-arena | grep executor

# gRPC 연결 테스트 (Backend Pod에서)
kubectl exec -it -n rl-arena rl-arena-backend-xxxxx-xxxxx -- /bin/sh
# telnet rl-arena-executor.rl-arena.svc.cluster.local 50051
```

### ConfigMap/Secret 변경 후 적용
```bash
# ConfigMap 업데이트
kubectl apply -f configmap.yaml

# Secret 업데이트
kubectl apply -f secret.yaml

# Pod 재시작 (Rolling Update)
kubectl rollout restart deployment/rl-arena-backend -n rl-arena

# 재시작 상태 확인
kubectl rollout status deployment/rl-arena-backend -n rl-arena
```

## 📊 모니터링

### 로그 확인
```bash
# Backend 로그 (모든 Pod)
kubectl logs -n rl-arena -l app=rl-arena-backend -f

# 특정 Pod 로그
kubectl logs -n rl-arena POD_NAME -f

# 최근 100줄
kubectl logs -n rl-arena POD_NAME --tail=100
```

### 리소스 사용량
```bash
# Pod 리소스 사용량
kubectl top pods -n rl-arena

# Node 리소스 사용량
kubectl top nodes
```

## 🧹 정리

### 전체 삭제
```bash
# Namespace 삭제 (모든 리소스 삭제)
kubectl delete namespace rl-arena
```

### 개별 리소스 삭제
```bash
kubectl delete -f ingress.yaml
kubectl delete -f deployment.yaml
kubectl delete -f service.yaml
kubectl delete -f redis-deployment.yaml
kubectl delete -f postgres-statefulset.yaml
kubectl delete -f secret.yaml
kubectl delete -f configmap.yaml
kubectl delete -f namespace.yaml
```

## 📝 다음 단계

1. ✅ Backend K8s 배포 완료
2. ⏭️ **TODO #2**: Executor Service 설정 및 gRPC 통신 테스트
3. ⏭️ **TODO #4**: Backend에 gRPC 클라이언트 구현
4. ⏭️ **TODO #8**: Executor Proto 컴파일
5. ⏭️ **TODO #6**: Kaniko를 사용한 Agent Docker 빌드 파이프라인

## 🔗 관련 문서

- [K8S_DEPLOYMENT_GUIDE.md](../../K8S_DEPLOYMENT_GUIDE.md) - 전체 K8s 아키텍처
- [K8S_INTEGRATION_SUMMARY.md](../../K8S_INTEGRATION_SUMMARY.md) - 통합 요약
- [SYSTEM_ANALYSIS.md](../../SYSTEM_ANALYSIS.md) - 시스템 분석 및 TODO 리스트
