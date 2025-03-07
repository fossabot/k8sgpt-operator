/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package resources

import (
	"context"
	err "errors"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	r1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// SyncOrDestroy enum create or destroy
type SyncOrDestroy int

const (
	SyncOp SyncOrDestroy = iota
	DestroyOp
	DeploymentName = "k8sgpt-deployment"
)

// GetService Create service for K8sGPT
func GetService(config v1alpha1.K8sGPT) (*corev1.Service, error) {
	// Create service
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8sgpt",
			Namespace: config.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:               config.Kind,
					Name:               config.Name,
					UID:                config.UID,
					APIVersion:         config.APIVersion,
					BlockOwnerDeletion: utils.PtrBool(true),
					Controller:         utils.PtrBool(true),
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": DeploymentName,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}

	return &service, nil
}

// GetServiceAccount Create Service Account for K8sGPT and bind it to K8sGPT role
func GetServiceAccount(config v1alpha1.K8sGPT) (*corev1.ServiceAccount, error) {
	// Create service account
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8sgpt",
			Namespace: config.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:               config.Kind,
					Name:               config.Name,
					UID:                config.UID,
					APIVersion:         config.APIVersion,
					BlockOwnerDeletion: utils.PtrBool(true),
					Controller:         utils.PtrBool(true),
				},
			},
		},
	}

	return &serviceAccount, nil
}

// GetClusterRoleBinding Create cluster role binding for K8sGPT
func GetClusterRoleBinding(config v1alpha1.K8sGPT) (*r1.ClusterRoleBinding, error) {

	// Create cluster role binding
	clusterRoleBinding := r1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8sgpt",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:               config.Kind,
					Name:               config.Name,
					UID:                config.UID,
					APIVersion:         config.APIVersion,
					BlockOwnerDeletion: utils.PtrBool(true),
					Controller:         utils.PtrBool(true),
				},
			},
		},
		Subjects: []r1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "k8sgpt",
				Namespace: config.Namespace,
			},
		},
		RoleRef: r1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "k8sgpt",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	return &clusterRoleBinding, nil
}

// GetClusterRole Create ClusterRole for K8sGPT with cluster read all
func GetClusterRole(config v1alpha1.K8sGPT) (*r1.ClusterRole, error) {

	// Create cluster role
	clusterRole := r1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8sgpt",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:               config.Kind,
					Name:               config.Name,
					UID:                config.UID,
					APIVersion:         config.APIVersion,
					BlockOwnerDeletion: utils.PtrBool(true),
					Controller:         utils.PtrBool(true),
				},
			},
		},
		Rules: []r1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				// This is necessary for the creation of integrations
				Verbs: []string{"create", "list", "get", "watch", "delete"},
			},
			// Allow creation of custom resources
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	return &clusterRole, nil
}

// GetDeployment Create deployment with the latest K8sGPT image
func GetDeployment(config v1alpha1.K8sGPT) (*appsv1.Deployment, error) {

	// Create deployment
	image := config.Spec.Repository + ":" + config.Spec.Version
	replicas := int32(1)
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName,
			Namespace: config.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:               config.Kind,
					Name:               config.Name,
					UID:                config.UID,
					APIVersion:         config.APIVersion,
					BlockOwnerDeletion: utils.PtrBool(true),
					Controller:         utils.PtrBool(true),
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": DeploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": DeploymentName,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "k8sgpt",
					Containers: []corev1.Container{
						{
							Name:            "k8sgpt",
							ImagePullPolicy: corev1.PullAlways,
							Image:           image,
							Args: []string{
								"serve",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "K8SGPT_MODEL",
									Value: config.Spec.AI.Model,
								},
								{
									Name:  "K8SGPT_BACKEND",
									Value: config.Spec.AI.Backend,
								},
								{
									Name:  "XDG_CONFIG_HOME",
									Value: "/k8sgpt-data/.config",
								},
								{
									Name:  "XDG_CACHE_HOME",
									Value: "/k8sgpt-data/.cache",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("0.2"),
									corev1.ResourceMemory: resource.MustParse("156Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/k8sgpt-data",
									Name:      "k8sgpt-vol",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
							Name:         "k8sgpt-vol",
						},
					},
				},
			},
		},
	}
	if config.Spec.AI.Secret != nil {
		password := corev1.EnvVar{
			Name: "K8SGPT_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: config.Spec.AI.Secret.Name,
					},
					Key: config.Spec.AI.Secret.Key,
				},
			},
		}
		deployment.Spec.Template.Spec.Containers[0].Env = append(
			deployment.Spec.Template.Spec.Containers[0].Env, password,
		)
	}
	if config.Spec.RemoteCache != nil {

		// check to see if key/value exists
		addRemoteCacheEnvVar := func(name, key string) {
			envVar := v1.EnvVar{
				Name: name,
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: config.Spec.RemoteCache.Credentials.Name,
						},
						Key: key,
					},
				},
			}
			deployment.Spec.Template.Spec.Containers[0].Env = append(
				deployment.Spec.Template.Spec.Containers[0].Env, envVar,
			)
		}
		if config.Spec.RemoteCache.Azure != nil {
			addRemoteCacheEnvVar("AZURE_CLIENT_ID", "azure_client_id")
			addRemoteCacheEnvVar("AZURE_TENANT_ID", "azure_tenant_id")
			addRemoteCacheEnvVar("AZURE_CLIENT_SECRET", "azure_client_secret")
		} else if config.Spec.RemoteCache.S3 != nil {
			addRemoteCacheEnvVar("AWS_ACCESS_KEY_ID", "aws_access_key_id")
			addRemoteCacheEnvVar("AWS_SECRET_ACCESS_KEY", "aws_secret_access_key")
		}
	}

	if config.Spec.AI.BaseUrl != "" {
		baseUrl := corev1.EnvVar{
			Name:  "K8SGPT_BASEURL",
			Value: config.Spec.AI.BaseUrl,
		}
		deployment.Spec.Template.Spec.Containers[0].Env = append(
			deployment.Spec.Template.Spec.Containers[0].Env, baseUrl,
		)
	}
	// Engine is required only when azureopenai is the ai backend
	if config.Spec.AI.Engine != "" && config.Spec.AI.Backend == v1alpha1.AzureOpenAI {
		engine := corev1.EnvVar{
			Name:  "K8SGPT_ENGINE",
			Value: config.Spec.AI.Engine,
		}
		deployment.Spec.Template.Spec.Containers[0].Env = append(
			deployment.Spec.Template.Spec.Containers[0].Env, engine,
		)
	} else if config.Spec.AI.Engine != "" && config.Spec.AI.Backend != v1alpha1.AzureOpenAI {
		return &appsv1.Deployment{}, err.New("Engine is supported only by azureopenai provider.")
	}
	return &deployment, nil
}

func Sync(ctx context.Context, c client.Client,
	config v1alpha1.K8sGPT, i SyncOrDestroy) error {

	var objs []client.Object

	svc, er := GetService(config)
	if er != nil {
		return er
	}

	objs = append(objs, svc)

	svcAcc, er := GetServiceAccount(config)
	if er != nil {
		return er
	}

	objs = append(objs, svcAcc)

	clusterRole, er := GetClusterRole(config)
	if er != nil {
		return er
	}

	objs = append(objs, clusterRole)

	clusterRoleBinding, er := GetClusterRoleBinding(config)
	if er != nil {
		return er
	}

	objs = append(objs, clusterRoleBinding)

	deployment, er := GetDeployment(config)
	if er != nil {
		return er
	}

	objs = append(objs, deployment)

	// for each object, create or destroy
	for _, obj := range objs {
		switch i {
		case SyncOp:

			// before creation, we will check to see if the secret exists if used as a ref
			if config.Spec.AI.Secret != nil {

				secret := &corev1.Secret{}
				er := c.Get(ctx, types.NamespacedName{Name: config.Spec.AI.Secret.Name,
					Namespace: config.Namespace}, secret)
				if er != nil {
					return err.New("references secret does not exist, cannot create deployment")
				}
			}

			err := doSync(ctx, c, obj)
			if err != nil {
				// If the object already exists, ignore the error
				if !errors.IsAlreadyExists(err) {
					return err
				}
			}
		case DestroyOp:
			err := c.Delete(ctx, obj)
			if err != nil {
				// if the object is not found, ignore the error
				if !errors.IsNotFound(err) {
					return err
				}
			}
		}
	}

	return nil
}

func doSync(ctx context.Context, clt client.Client, obj client.Object) error {
	var mutateFn controllerutil.MutateFn
	switch expect := obj.(type) {
	case *appsv1.Deployment:
		exist := &appsv1.Deployment{}
		err := clt.Get(context.Background(), client.ObjectKeyFromObject(obj), exist)
		if err != nil && !errors.IsNotFound(err) {
			return err
		} else if err == nil {
			mutateFn = func() error {
				exist.Spec = expect.Spec
				return nil
			}
			obj = exist
		}
	case *corev1.Service:
		exist := &corev1.Service{}
		err := clt.Get(context.Background(), client.ObjectKeyFromObject(obj), exist)
		if err != nil && !errors.IsNotFound(err) {
			return err
		} else if err == nil {
			mutateFn = func() error {
				exist.Spec = expect.Spec
				return nil
			}
			obj = exist
		}
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := controllerutil.CreateOrPatch(ctx, clt, obj, mutateFn)
		return err
	})
}
