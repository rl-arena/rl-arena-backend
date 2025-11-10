package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// BuilderService Kaniko를 사용하여 Docker 이미지 빌드
type BuilderService struct {
	k8sClient       *kubernetes.Clientset
	submissionRepo  *repository.SubmissionRepository
	namespace       string
	registryURL     string
	registrySecret  string
	useLocalK8s     bool // 로컬 개발 환경 여부
}

// NewBuilderService BuilderService 생성
func NewBuilderService(
	submissionRepo *repository.SubmissionRepository,
	namespace, registryURL, registrySecret string,
) (*BuilderService, error) {
	var config *rest.Config
	var err error

	// 로컬 K8s 사용 여부 확인
	useLocalK8s := os.Getenv("USE_LOCAL_K8S") == "true"

	if useLocalK8s {
		// 로컬 kubeconfig 사용 (Docker Desktop K8s)
		kubeconfig := filepath.Join(os.Getenv("USERPROFILE"), ".kube", "config")
		if os.Getenv("KUBECONFIG") != "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	} else {
		// K8s in-cluster 설정 (프로덕션)
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &BuilderService{
		k8sClient:      clientset,
		submissionRepo: submissionRepo,
		namespace:      namespace,
		registryURL:    registryURL,
		registrySecret: registrySecret,
		useLocalK8s:    useLocalK8s,
	}, nil
}

// BuildAgentImage Agent 코드를 Docker 이미지로 빌드
func (s *BuilderService) BuildAgentImage(ctx context.Context, submission *models.Submission) error {
	// Docker 이미지 태그 생성
	imageTag := fmt.Sprintf("%s/%s:%s-v%d",
		s.registryURL,
		submission.AgentID,
		submission.ID,
		submission.Version,
	)

	// Kaniko Job 생성
	jobName := fmt.Sprintf("build-%s", submission.ID)
	job := s.createKanikoJob(jobName, submission.CodeURL, imageTag, submission.ID)

	// Job 생성
	createdJob, err := s.k8sClient.BatchV1().Jobs(s.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kaniko job: %w", err)
	}

	// Submission에 Job 정보 저장
	buildJobName := createdJob.Name
	submission.BuildJobName = &buildJobName
	submission.DockerImageURL = &imageTag

	// Job의 첫 번째 Pod 이름 가져오기 (비동기로 생성되므로 대기)
	podName, err := s.waitForPod(ctx, jobName)
	if err == nil && podName != "" {
		submission.BuildPodName = &podName
	}

	return nil
}

// createKanikoJob Kaniko Job YAML 생성
func (s *BuilderService) createKanikoJob(jobName, codeURL, imageTag, submissionID string) *batchv1.Job {
	backoffLimit := int32(3)
	ttlSecondsAfterFinished := int32(3600) // 1시간 후 자동 삭제

	// 로컬 환경: Docker-in-Docker 사용
	if s.useLocalK8s {
		return s.createDockerBuildJob(jobName, codeURL, imageTag, submissionID, backoffLimit, ttlSecondsAfterFinished)
	}

	// 프로덕션 환경: Kaniko 사용
	return s.createKanikoBuildJob(jobName, codeURL, imageTag, submissionID, backoffLimit, ttlSecondsAfterFinished)
}

// createDockerBuildJob 로컬 개발용 Docker-in-Docker Job 생성
func (s *BuilderService) createDockerBuildJob(jobName, codeURL, imageTag, submissionID string, backoffLimit, ttlSecondsAfterFinished int32) *batchv1.Job {
	// URL 기반인지 로컬 파일인지 확인
	isGitRepo := strings.HasPrefix(codeURL, "http://") || strings.HasPrefix(codeURL, "https://")
	
	var initContainers []corev1.Container
	
	if isGitRepo {
		// Git 저장소에서 코드 clone
		initContainers = append(initContainers, corev1.Container{
			Name:  "git-clone",
			Image: "alpine/git:latest",
			Command: []string{
				"sh", "-c",
				fmt.Sprintf("git clone %s /workspace", codeURL),
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: "/workspace",
				},
			},
		})
	} else {
		// 로컬 파일 - placeholder 코드 생성
		initContainers = append(initContainers, corev1.Container{
			Name:  "create-code",
			Image: "busybox:latest",
			Command: []string{
				"sh", "-c",
				`cat > /workspace/agent.py << 'EOF'
# Placeholder agent code
import gymnasium as gym
env = gym.make('CartPole-v1')
EOF
cat > /workspace/Dockerfile << 'EOF'
FROM python:3.10-slim
WORKDIR /app
RUN pip install gymnasium
COPY agent.py /app/
CMD ["python", "agent.py"]
EOF`,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: "/workspace",
				},
			},
		})
	}
	
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app":            "rl-arena",
				"type":           "agent-build",
				"submission-id":  submissionID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  "rl-arena",
						"type": "agent-build",
						"job":  jobName,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyNever,
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:  "docker-build",
							Image: "docker:24-dind",
							Command: []string{
								"sh", "-c",
								fmt.Sprintf(`
									# Docker 데몬 시작 대기
									dockerd-entrypoint.sh &
									sleep 5
									
									# 이미지 빌드
									cd /workspace
									docker build -t %s .
									
									echo "Build completed successfully"
								`, imageTag),
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: func() *bool { b := true; return &b }(),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workspace",
									MountPath: "/workspace",
								},
								{
									Name:      "docker-graph-storage",
									MountPath: "/var/lib/docker",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "docker-graph-storage",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}

// createKanikoBuildJob 프로덕션용 Kaniko Job 생성
func (s *BuilderService) createKanikoBuildJob(jobName, codeURL, imageTag, submissionID string, backoffLimit, ttlSecondsAfterFinished int32) *batchv1.Job {

	// URL 기반인지 로컬 파일인지 확인
	isGitRepo := strings.HasPrefix(codeURL, "http://") || strings.HasPrefix(codeURL, "https://")
	
	var initContainers []corev1.Container
	var volumes []corev1.Volume
	
	// Workspace volume (항상 필요)
	volumes = append(volumes, corev1.Volume{
		Name: "workspace",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
	
	// Kaniko secret volume (항상 필요)
	volumes = append(volumes, corev1.Volume{
		Name: "kaniko-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: s.registrySecret,
				Items: []corev1.KeyToPath{
					{
						Key:  ".dockerconfigjson",
						Path: "config.json",
					},
				},
			},
		},
	})
	
	if isGitRepo {
		// Git 저장소에서 코드 clone
		initContainers = append(initContainers, corev1.Container{
			Name:  "git-clone",
			Image: "alpine/git:latest",
			Command: []string{
				"sh", "-c",
				fmt.Sprintf("git clone %s /workspace", codeURL),
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: "/workspace",
				},
			},
		})
	} else {
		// 로컬 파일 경로 - ConfigMap 또는 간단한 echo로 생성
		// TODO: 실제로는 hostPath나 PVC 사용 권장
		initContainers = append(initContainers, corev1.Container{
			Name:  "create-code",
			Image: "busybox:latest",
			Command: []string{
				"sh", "-c",
				`cat > /workspace/agent.py << 'EOF'
# Placeholder agent code
import gym
env = gym.make('Pong-v0')
EOF
cat > /workspace/Dockerfile << 'EOF'
FROM python:3.10-slim
WORKDIR /app
RUN pip install gymnasium[atari]
COPY agent.py /app/
CMD ["python", "agent.py"]
EOF`,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "workspace",
					MountPath: "/workspace",
				},
			},
		})
	}
	
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app":            "rl-arena",
				"type":           "agent-build",
				"submission-id":  submissionID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  "rl-arena",
						"type": "agent-build",
						"job":  jobName,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyNever,
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:  "kaniko",
							Image: "gcr.io/kaniko-project/executor:latest",
							Args: []string{
								"--dockerfile=/workspace/Dockerfile",
								"--context=/workspace",
								fmt.Sprintf("--destination=%s", imageTag),
								"--cache=true",
								"--cache-ttl=24h",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workspace",
									MountPath: "/workspace",
								},
								{
									Name:      "kaniko-secret",
									MountPath: "/kaniko/.docker",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "DOCKER_CONFIG",
									Value: "/kaniko/.docker",
								},
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}

// waitForPod Job의 Pod가 생성될 때까지 대기
func (s *BuilderService) waitForPod(ctx context.Context, jobName string) (string, error) {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for pod")
		case <-ticker.C:
			pods, err := s.k8sClient.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("job=%s", jobName),
			})
			if err != nil {
				continue
			}
			if len(pods.Items) > 0 {
				return pods.Items[0].Name, nil
			}
		}
	}
}

// GetBuildStatus Job의 빌드 상태 확인
func (s *BuilderService) GetBuildStatus(ctx context.Context, jobName string) (string, error) {
	job, err := s.k8sClient.BatchV1().Jobs(s.namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get job: %w", err)
	}

	if job.Status.Succeeded > 0 {
		return "succeeded", nil
	}
	if job.Status.Failed > 0 {
		return "failed", nil
	}
	if job.Status.Active > 0 {
		return "building", nil
	}

	return "pending", nil
}

// GetBuildLogs Pod의 빌드 로그 가져오기
func (s *BuilderService) GetBuildLogs(ctx context.Context, podName string) (string, error) {
	req := s.k8sClient.CoreV1().Pods(s.namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: "kaniko",
	})

	logs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer logs.Close()

	buf := make([]byte, 10000) // 로그 크기 제한
	n, _ := logs.Read(buf)

	return string(buf[:n]), nil
}
