package main

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestMatchesDanglingCriteria(t *testing.T) {
	// Setup standard selectors
	activeSelectors := []labels.Selector{
		labels.SelectorFromSet(map[string]string{"app": "web"}),
	}

	now := time.Now()

	tests := []struct {
		name          string
		pod           corev1.Pod
		minAge        time.Duration
		includeKubeNs bool
		want          bool
	}{
		{
			name: "Dangling pod (No service match, old enough)",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "orphan-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					Labels:            map[string]string{"app": "unmatched"},
				},
			},
			minAge:        time.Hour,
			includeKubeNs: false,
			want:          true,
		},
		{
			name: "Not dangling (Matched by service)",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "web-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					Labels:            map[string]string{"app": "web"},
				},
			},
			minAge:        time.Hour,
			includeKubeNs: false,
			want:          false,
		},
		{
			name: "Skip because too young",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "new-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(now.Add(-10 * time.Minute)),
				},
			},
			minAge:        time.Hour,
			includeKubeNs: false,
			want:          false,
		},
		{
			name: "Skip because of annotation",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "protected-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					Annotations:       map[string]string{SKIP_ANNOTATION_NAME: SKIP_ANNOTATION_VALUE},
				},
			},
			minAge:        time.Hour,
			includeKubeNs: false,
			want:          false,
		},
		{
			name: "Skip kube-system pod by default",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "kube-proxy",
					Namespace:         "kube-system",
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
				},
			},
			minAge:        time.Hour,
			includeKubeNs: false,
			want:          false,
		},
		{
			name: "Include kube-system pod when flag is true",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "kube-proxy",
					Namespace:         "kube-system",
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
				},
			},
			minAge:        time.Hour,
			includeKubeNs: true,
			want:          true,
		},
		{
			name: "Skip Job pods via OwnerReference",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pi-job-abcde",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Job", Name: "pi-job"},
					},
				},
			},
			minAge:        time.Hour,
			includeKubeNs: false,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesDanglingCriteria(tt.pod, activeSelectors, tt.minAge, tt.includeKubeNs)
			if got != tt.want {
				t.Errorf("matchesDanglingCriteria() = %v, want %v", got, tt.want)
			}
		})
	}
}
