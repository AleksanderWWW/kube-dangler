package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const VERSION string = "0.1.0"

const SKIP_ANNOTATION_NAME string = "kubedangler/skip"
const SKIP_ANNOTATION_VALUE string = "true"

// matchesDanglingCriteria evaluates if a pod is "dangling" based on the provided configuration.
// Returns true if the pod could potentially be considered dangling (unattached/orphan).
func matchesDanglingCriteria(pod corev1.Pod, activeSelectors []labels.Selector, minAge time.Duration, includeKubeNs bool) bool {
	// 1. Skip checking pods from namespaces like kube-system, kube-public etc. unless required
	if !includeKubeNs && strings.HasPrefix(pod.Namespace, "kube-") {
		return false
	}

	// 2. Skip pods with special annotation
	if val, ok := pod.Annotations[SKIP_ANNOTATION_NAME]; ok && val == SKIP_ANNOTATION_VALUE {
		return false
	}

	// 3. Skip pods that are younger than minAge
	if time.Since(pod.CreationTimestamp.Time) < minAge {
		return false
	}

	// 4. Skip pods that are part of a Job (Jobs are expected to be short-lived/unattached)
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "Job" {
			return false
		}
	}

	// 5. Check if any active Service selector matches this pod
	for _, selector := range activeSelectors {
		if selector.Matches(labels.Set(pod.Labels)) {
			return false // It's matched to a service, so it's not dangling
		}
	}

	return true
}

func fetchDanglers(ctx context.Context, namespace string, minAge time.Duration, includeKubeNs bool) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		home, err := homeDir()
		if err != nil {
			return err
		}
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

		if err != nil {
			return err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	services, _ := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})

	activeSelectors := []labels.Selector{}

	for _, svc := range services.Items {
		if len(svc.Spec.Selector) > 0 {
			activeSelectors = append(activeSelectors, labels.SelectorFromSet(svc.Spec.Selector))
		}
	}

	for _, pod := range pods.Items {
		isMatched := matchesDanglingCriteria(pod, activeSelectors, minAge, includeKubeNs)
		if !isMatched {
			fmt.Printf("[%s] %-20s (Age: %s)\n",
				pod.Namespace, pod.Name, time.Since(pod.CreationTimestamp.Time).Round(time.Second))
		}
	}
	return nil
}

func homeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory")
	}

	return home, nil
}

func main() {
	cmd := &cli.Command{
		Name:  "kubedangler",
		Usage: "find potentially dangling Pods (attached to no Service)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Value:   "",
				Usage:   "namespace to check for dangling pods (default: look through all namespaces)",
			},
			&cli.DurationFlag{
				Name:  "min-age",
				Value: time.Hour,
				Usage: "minimal age of potentially dangling pods",
			},
			&cli.BoolFlag{
				Name:  "include-kube-ns",
				Usage: "whether to also include checking the kube namespaces",
			},
			&cli.BoolFlag{
				Name:  "version",
				Usage: "print version number and exit",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			version := cmd.Bool("version")
			if version {
				fmt.Println(VERSION)
				return nil
			}
			namespace := cmd.String("namespace")
			minAge := cmd.Duration("min-age")
			includeKubeNs := cmd.Bool("include-kube-ns")
			return fetchDanglers(ctx, namespace, minAge, includeKubeNs)
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
