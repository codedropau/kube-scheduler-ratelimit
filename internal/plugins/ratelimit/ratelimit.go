package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientsetcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

const (
	// AnnotationLimit is used to get the rate limit a Pod from being scheduled.
	AnnotationLimit = "kube-scheduler-ratelimit/limit"
	// AnnotationQuery is used to get the label query used to rate limit a Pod from being scheduled.
	AnnotationQuery = "kube-scheduler-ratelimit/query"
	// AnnotationScheduled is used to determine if a Pod is scheduled.
	// It is also used to determine when a Pod is scheduled.
	AnnotationScheduled = "kube-scheduler-ratelimit/scheduled"
)

// Plugin that checks if a pod spec node name matches the current node.
type Plugin struct{
	clientset clientsetcorev1.CoreV1Interface
}

var _ framework.PermitPlugin = &Plugin{}

// Name of this Scheduler.
const Name = "RateLimit"

// Name returns name of the plugin. It is used in logs, etc.
func (pl *Plugin) Name() string {
	return Name
}

// Permit are used to delay Pods from being scheduled.
// https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/20180409-scheduling-framework.md#permit
func (pl *Plugin) Permit(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) (*framework.Status, time.Duration) {
	retry := time.Second * 15

	query, limit, err := pl.GetAnnotations(pod)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error()), retry
	}

	running, err := pl.CheckLimit(ctx, query)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error()), retry
	}

	count := len(running) + 1

	if count > limit {
		return framework.NewStatus(framework.Wait, fmt.Sprintf("rate limiting has been triggered, query %s returns %d running Pods", query, count)), retry
	}

	err = pl.TagPod(ctx, pod)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error()), retry
	}

	return framework.NewStatus(framework.Success, ""), retry
}

// GetAnnotations will return the limit and query used when scheduling.
func (pl *Plugin) GetAnnotations(pod *corev1.Pod) (string, int, error) {
	l, err := getAnnotationValue(pod.ObjectMeta.Annotations, AnnotationLimit)
	if err != nil {
		return "", 0, err
	}

	limit, err := strconv.Atoi(l)
	if err != nil {
		return "", 0, err
	}

	query, err := getAnnotationValue(pod.ObjectMeta.Annotations, AnnotationQuery)
	if err != nil {
		return "", 0, err
	}

	return query, limit, nil
}
// CheckLimit will perform the query and determine how many Pods are running.
// You might notice we don't query for Pods which a Phase of "Running", this is because we have to cover edge cases eg.
//   Pods = Pending + Has already been scheduled by this plugin.
func (pl *Plugin) CheckLimit(ctx context.Context, query string) ([]corev1.Pod, error) {
	var pods []corev1.Pod

	list, err := pl.clientset.Pods(corev1.NamespaceAll).List(ctx, metav1.ListOptions{
		LabelSelector: query,
	})
	if err != nil {
		return pods, err
	}

	for _, item := range list.Items {
		if item.Status.Phase == corev1.PodSucceeded {
			continue
		}

		if item.Status.Phase == corev1.PodFailed {
			continue
		}

		if item.Status.Phase == corev1.PodUnknown {
			continue
		}

		// If it hasn't been tagged with an annotation yet then it is not classified as "running".
		if _, ok := item.ObjectMeta.Annotations[AnnotationScheduled]; !ok {
			continue
		}

		pods = append(pods, item)
	}

	return pods, nil
}

// TagPod with so it is marked as "scheduled" by this plugin.
func (pl *Plugin) TagPod(ctx context.Context, pod *corev1.Pod) error {
	// We don't need to double tag.
	if _, ok := pod.ObjectMeta.Annotations[AnnotationScheduled]; ok {
		return nil
	}

	oldData, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	pod.ObjectMeta.Annotations[AnnotationScheduled] = time.Now().String()

	newData, err := json.Marshal(pod)
	createdPatch := err == nil
	if err != nil {
		return err
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	if !createdPatch {
		return nil
	}

	_, err = pl.clientset.Pods(pod.ObjectMeta.Namespace).Patch(ctx, pod.ObjectMeta.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

// New initializes a new plugin and returns it.
func New(_ *runtime.Unknown, handle framework.FrameworkHandle) (framework.Plugin, error) {
	return &Plugin{handle.ClientSet().CoreV1()}, nil
}

// Helper function to get an annotation and return an error when not found.
func getAnnotationValue(annotations map[string]string, key string) (string, error) {
	if val, ok := annotations[key]; ok {
		return val, nil
	}

	return "", fmt.Errorf("annotation not found: %s", key)
}