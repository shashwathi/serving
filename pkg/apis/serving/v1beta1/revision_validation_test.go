/*
Copyright 2019 The Knative Authors

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

package v1beta1

import (
	"context"
	"fmt"
	"testing"

	"knative.dev/serving/pkg/apis/autoscaling"
	"knative.dev/serving/pkg/apis/config"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/ptr"
)

func TestRevisionValidation(t *testing.T) {
	tests := []struct {
		name string
		r    *Revision
		want *apis.FieldError
	}{{
		name: "valid",
		r: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "busybox",
					}},
				},
			},
		},
		want: nil,
	}, {
		name: "invalid container concurrency",
		r: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "busybox",
					}},
				},
				ContainerConcurrency: -10,
			},
		},
		want: apis.ErrOutOfBoundsValue(
			-10, 0, RevisionContainerConcurrencyMax,
			"spec.containerConcurrency"),
	}}

	// TODO(dangerd): PodSpec validation failures.

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.r.Validate(context.Background())
			if !cmp.Equal(test.want.Error(), got.Error()) {
				t.Errorf("Validate (-want, +got) = %v",
					cmp.Diff(test.want.Error(), got.Error()))
			}
		})
	}
}

func TestRevisionLabelValidation(t *testing.T) {
	validRevisionSpec := RevisionSpec{
		PodSpec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Image: "busybox",
			}},
		},
	}
	tests := []struct {
		name string
		r    *Revision
		want *apis.FieldError
	}{{
		name: "valid route name",
		r: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "byo-name",
				Labels: map[string]string{
					"serving.knative.dev/route": "test-route",
				},
			},
			Spec: validRevisionSpec,
		},
		want: nil,
	}, {
		name: "valid knative service name",
		r: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "byo-name",
				Labels: map[string]string{
					"serving.knative.dev/service": "test-svc",
				},
			},
			Spec: validRevisionSpec,
		},
		want: nil,
	}, {
		name: "valid knative service name",
		r: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "byo-name",
				Labels: map[string]string{
					"serving.knative.dev/configurationGeneration": "1234",
				},
			},
			Spec: validRevisionSpec,
		},
		want: nil,
	}, {
		name: "invalid knative configuration name",
		r: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "byo-name",
				Labels: map[string]string{
					"serving.knative.dev/configuration": "absent-cfg",
				},
			},
			Spec: validRevisionSpec,
		},
		want: apis.ErrInvalidValue("absent-cfg", "metadata.label.[serving.knative.dev/configuration]"),
	}, {
		name: "valid knative configuration name",
		r: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "byo-name",
				Labels: map[string]string{
					"serving.knative.dev/configuration": "test-cfg",
				},
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "serving.knative.dev/v1alpha1",
					Kind:       "Configuration",
					Name:       "test-cfg",
				}},
			},
			Spec: validRevisionSpec,
		},
		want: nil,
	}, {
		name: "invalid knative label",
		r: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "byo-name",
				Labels: map[string]string{
					"serving.knative.dev/testlabel": "value",
				},
			},
			Spec: validRevisionSpec,
		},
		want: apis.ErrInvalidKeyName("serving.knative.dev/testlabel", "metadata.label"),
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.r.Validate(context.Background())
			if !cmp.Equal(test.want.Error(), got.Error()) {
				t.Errorf("Validate (-want, +got) = %v",
					cmp.Diff(test.want.Error(), got.Error()))
			}
		})
	}
}

func TestContainerConcurrencyValidation(t *testing.T) {
	tests := []struct {
		name string
		cc   RevisionContainerConcurrencyType
		want *apis.FieldError
	}{{
		name: "single",
		cc:   1,
		want: nil,
	}, {
		name: "unlimited",
		cc:   0,
		want: nil,
	}, {
		name: "ten",
		cc:   10,
		want: nil,
	}, {
		name: "invalid container concurrency (too small)",
		cc:   -1,
		want: apis.ErrOutOfBoundsValue(-1, 0, RevisionContainerConcurrencyMax,
			apis.CurrentField),
	}, {
		name: "invalid container concurrency (too large)",
		cc:   RevisionContainerConcurrencyMax + 1,
		want: apis.ErrOutOfBoundsValue(RevisionContainerConcurrencyMax+1,
			0, RevisionContainerConcurrencyMax, apis.CurrentField),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.cc.Validate(context.Background())
			if !cmp.Equal(test.want.Error(), got.Error()) {
				t.Errorf("Validate (-want, +got) = %v",
					cmp.Diff(test.want.Error(), got.Error()))
			}
		})
	}
}

func TestRevisionSpecValidation(t *testing.T) {
	tests := []struct {
		name string
		rs   *RevisionSpec
		wc   func(context.Context) context.Context
		want *apis.FieldError
	}{{
		name: "valid",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "helloworld",
				}},
			},
		},
		want: nil,
	}, {
		name: "with volume (ok)",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "helloworld",
					VolumeMounts: []corev1.VolumeMount{{
						MountPath: "/mount/path",
						Name:      "the-name",
						ReadOnly:  true,
					}},
				}},
				Volumes: []corev1.Volume{{
					Name: "the-name",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "foo",
						},
					},
				}},
			},
		},
		want: nil,
	}, {
		name: "with volume name collision",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "helloworld",
					VolumeMounts: []corev1.VolumeMount{{
						MountPath: "/mount/path",
						Name:      "the-name",
						ReadOnly:  true,
					}},
				}},
				Volumes: []corev1.Volume{{
					Name: "the-name",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "foo",
						},
					},
				}, {
					Name: "the-name",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{},
					},
				}},
			},
		},
		want: (&apis.FieldError{
			Message: fmt.Sprintf(`duplicate volume name "the-name"`),
			Paths:   []string{"name"},
		}).ViaFieldIndex("volumes", 1),
	}, {
		name: "bad pod spec",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:      "steve",
					Image:     "helloworld",
					Lifecycle: &corev1.Lifecycle{},
				}},
			},
		},
		want: apis.ErrDisallowedFields("containers[0].lifecycle"),
	}, {
		name: "missing container",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{},
			},
		},
		want: apis.ErrMissingField("containers"),
	}, {
		name: "too many containers",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "busybox",
				}, {
					Image: "helloworld",
				}},
			},
		},
		want: apis.ErrMultipleOneOf("containers"),
	}, {
		name: "exceed max timeout",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "helloworld",
				}},
			},
			TimeoutSeconds: ptr.Int64(6000),
		},
		want: apis.ErrOutOfBoundsValue(
			6000, 0, config.DefaultMaxRevisionTimeoutSeconds,
			"timeoutSeconds"),
	}, {
		name: "exceed custom max timeout",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "helloworld",
				}},
			},
			TimeoutSeconds: ptr.Int64(100),
		},
		wc: func(ctx context.Context) context.Context {
			s := config.NewStore(logtesting.TestLogger(t))
			s.OnConfigChanged(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: config.DefaultsConfigName,
				},
				Data: map[string]string{
					"revision-timeout-seconds":     "25",
					"max-revision-timeout-seconds": "50"},
			})
			return s.ToContext(ctx)
		},
		want: apis.ErrOutOfBoundsValue(100, 0, 50, "timeoutSeconds"),
	}, {
		name: "negative timeout",
		rs: &RevisionSpec{
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "helloworld",
				}},
			},
			TimeoutSeconds: ptr.Int64(-30),
		},
		want: apis.ErrOutOfBoundsValue(
			-30, 0, config.DefaultMaxRevisionTimeoutSeconds,
			"timeoutSeconds"),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.wc != nil {
				ctx = test.wc(ctx)
			}
			got := test.rs.Validate(ctx)
			if !cmp.Equal(test.want.Error(), got.Error()) {
				t.Errorf("Validate (-want, +got) = %v",
					cmp.Diff(test.want.Error(), got.Error()))
			}
		})
	}
}

func TestImmutableFields(t *testing.T) {
	tests := []struct {
		name string
		new  *Revision
		old  *Revision
		wc   func(context.Context) context.Context
		want *apis.FieldError
	}{{
		name: "good (no change)",
		new: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
			},
		},
		old: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
			},
		},
		want: nil,
	}, {
		// Test the case where max-revision-timeout is changed to a value
		// that is less than an existing revision's timeout value.
		// Existing revision should keep operating normally.
		name: "good (max revision timeout change)",
		new: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
				TimeoutSeconds: ptr.Int64(100),
			},
		},
		old: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
				TimeoutSeconds: ptr.Int64(100),
			},
		},
		wc: func(ctx context.Context) context.Context {
			s := config.NewStore(logtesting.TestLogger(t))
			s.OnConfigChanged(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: config.DefaultsConfigName,
				},
				Data: map[string]string{
					"revision-timeout-seconds":     "25",
					"max-revision-timeout-seconds": "50"},
			})
			return s.ToContext(ctx)
		},
		want: nil,
	}, {
		name: "bad (resources image change)",
		new: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName("cpu"): resource.MustParse("50m"),
							},
						},
					}},
				},
			},
		},
		old: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName("cpu"): resource.MustParse("100m"),
							},
						},
					}},
				},
			},
		},
		want: &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: `{v1beta1.RevisionSpec}.PodSpec.Containers[0].Resources.Requests["cpu"]:
	-: resource.Quantity: "{i:{value:100 scale:-3} d:{Dec:<nil>} s:100m Format:DecimalSI}"
	+: resource.Quantity: "{i:{value:50 scale:-3} d:{Dec:<nil>} s:50m Format:DecimalSI}"
`,
		},
	}, {
		name: "bad (container image change)",
		new: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
			},
		},
		old: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "busybox",
					}},
				},
			},
		},
		want: &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: `{v1beta1.RevisionSpec}.PodSpec.Containers[0].Image:
	-: "busybox"
	+: "helloworld"
`,
		},
	}, {
		name: "bad (concurrency model change)",
		new: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
				ContainerConcurrency: 1,
			},
		},
		old: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
				ContainerConcurrency: 2,
			},
		},
		want: &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: `{v1beta1.RevisionSpec}.ContainerConcurrency:
	-: "2"
	+: "1"
`,
		},
	}, {
		name: "bad (new field added)",
		new: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
				ContainerConcurrency: 42,
			},
		},
		old: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
			},
		},
		want: &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: `{v1beta1.RevisionSpec}.ContainerConcurrency:
	-: "0"
	+: "42"
`,
		},
	}, {
		name: "bad (multiple changes)",
		new: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
				ContainerConcurrency: 2,
			},
		},
		old: &Revision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "valid",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "busybox",
					}},
				},
				ContainerConcurrency: 4,
			},
		},
		want: &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: `{v1beta1.RevisionSpec}.PodSpec.Containers[0].Image:
	-: "busybox"
	+: "helloworld"
{v1beta1.RevisionSpec}.ContainerConcurrency:
	-: "4"
	+: "2"
`,
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := apis.WithinUpdate(context.Background(), test.old)
			if test.wc != nil {
				ctx = test.wc(ctx)
			}
			got := test.new.Validate(ctx)
			if got, want := got.Error(), test.want.Error(); got != want {
				t.Errorf("Validate got: %s, want: %s, diff:(-want, +got)=\n%v", got, want, cmp.Diff(got, want))
			}
		})
	}
}

func TestRevisionTemplateSpecValidation(t *testing.T) {
	tests := []struct {
		name string
		rts  *RevisionTemplateSpec
		want *apis.FieldError
	}{{
		name: "valid",
		rts: &RevisionTemplateSpec{
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
			},
		},
		want: nil,
	}, {
		name: "empty spec",
		rts:  &RevisionTemplateSpec{},
		want: apis.ErrMissingField("spec.containers"),
	}, {
		name: "nested spec error",
		rts: &RevisionTemplateSpec{
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:      "kevin",
						Image:     "helloworld",
						Lifecycle: &corev1.Lifecycle{},
					}},
				},
			},
		},
		want: apis.ErrDisallowedFields("spec.containers[0].lifecycle"),
	}, {
		name: "has revision template name",
		rts: &RevisionTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				// We let users bring their own revision name.
				Name: "parent-foo",
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
			},
		},
		want: nil,
	}, {
		name: "invalid metadata.annotations for scale",
		rts: &RevisionTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					autoscaling.MinScaleAnnotationKey: "5",
					autoscaling.MaxScaleAnnotationKey: "",
				},
			},
			Spec: RevisionSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "helloworld",
					}},
				},
			},
		},
		want: (&apis.FieldError{
			Message: "expected 1 <=  <= 2147483647",
			Paths:   []string{autoscaling.MaxScaleAnnotationKey},
		}).ViaField("annotations").ViaField("metadata"),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := apis.WithinParent(context.Background(), metav1.ObjectMeta{
				Name: "parent",
			})

			got := test.rts.Validate(ctx)
			if diff := cmp.Diff(test.want.Error(), got.Error()); diff != "" {
				t.Errorf("Validate (-want, +got) = %v", diff)
			}
		})
	}
}
