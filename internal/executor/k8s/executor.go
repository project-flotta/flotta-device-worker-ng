package k8s

import (
	"context"
	"errors"
	"fmt"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/executor/common"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sExecutor struct {
	client *kubernetes.Clientset
}

func New(kubeConfigPath string) (*K8sExecutor, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s config from kubeconfig '%s': %w", kubeConfigPath, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s executor: %w", err)
	}
	return &K8sExecutor{client: clientset}, nil
}

func (e *K8sExecutor) Run(ctx context.Context, w entity.Workload) error {
	workload := w.(entity.PodWorkload)

	pod, err := common.ToPod(workload)
	if err != nil {
		zap.S().Errorw("failed to create pod", "error", err)
		return err
	}

	deploymentClient := e.client.AppsV1().Deployments(workload.Namespace)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: w.ID(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"id": w.ID()[:12],
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"id": w.ID()[:12],
					},
				},
				Spec: pod.Spec,
			},
		},
	}

	results, err := deploymentClient.Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		zap.S().Error(err)
		return err
	}
	zap.S().Debugw("deployment created", "job_id", w.ID(), "deployment_results", results)
	return nil
}

func (e *K8sExecutor) Exists(ctx context.Context, id string) (bool, error) {
	return false, errors.New("not implemented")
}

func (e *K8sExecutor) Start(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func (e *K8sExecutor) Stop(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func (e *K8sExecutor) Remove(ctx context.Context, w entity.Workload) error {
	workload := w.(entity.PodWorkload)
	deploymentsClient := e.client.AppsV1().Deployments(workload.Namespace)
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(ctx, workload.ID(), metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return err
	}
	return nil
}

func (e *K8sExecutor) List(ctx context.Context) ([]common.WorkloadInfo, error) {
	return []common.WorkloadInfo{}, nil
}

func (e *K8sExecutor) GetState(ctx context.Context, w entity.Workload) (entity.JobState, error) {
	workload := w.(entity.PodWorkload)
	deploymentClient := e.client.AppsV1().Deployments(workload.Namespace)
	list, err := deploymentClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return entity.UnknownState, err
	}
	for _, d := range list.Items {
		if d.ObjectMeta.Name == workload.ID() {
			return entity.RunningState, nil
		}
	}
	return entity.UnknownState, nil
}

func int32Ptr(i int32) *int32 { return &i }
