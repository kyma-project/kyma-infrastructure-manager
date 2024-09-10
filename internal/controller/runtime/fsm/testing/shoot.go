package testing

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ShootNoDNS = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
	}

	RuntimeOnlyName = v1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
			Labels: map[string]string{
				"kyma-project.io/instance-id":         "instance-id",
				"kyma-project.io/runtime-id":          "runtime-id",
				"kyma-project.io/shoot-name":          "shoot-name",
				"kyma-project.io/region":              "region",
				"operator.kyma-project.io/kyma-name":  "kyma-name",
				"kyma-project.io/broker-plan-id":      "broker-plan-id",
				"kyma-project.io/broker-plan-name":    "broker-plan-name",
				"kyma-project.io/global-account-id":   "global-account-id",
				"kyma-project.io/subaccount-id":       "subaccount-id",
				"operator.kyma-project.io/managed-by": "managed-by",
				"operator.kyma-project.io/internal":   "false",
				"kyma-project.io/platform-region":     "platform-region",
			},
		},
		Spec: v1.RuntimeSpec{
			Shoot: v1.RuntimeShoot{Name: "test-shoot"},
		},
	}

	ShootNoDNSDomain = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{},
		},
	}

	ShootMissingLastOperation = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{},
	}

	ShootEmptyLastOperation = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{},
		},
	}

	ShootLastOperationProcessing = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				Type:  gardener.LastOperationTypeCreate,
				State: gardener.LastOperationStateProcessing,
			},
		},
	}

	ShootLastOperationPending = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				Type:  gardener.LastOperationTypeCreate,
				State: gardener.LastOperationStatePending,
			},
		},
	}

	ShootLastOperationSucceeded = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				Type:  gardener.LastOperationTypeCreate,
				State: gardener.LastOperationStateSucceeded,
			},
		},
	}

	ShootLastOperationFailed = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				Type:  gardener.LastOperationTypeCreate,
				State: gardener.LastOperationStateFailed,
			},
		},
	}

	ShootLastOperationReconcileProcessing = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				Type:  gardener.LastOperationTypeReconcile,
				State: gardener.LastOperationStateProcessing,
			},
		},
	}

	ShootLastOperationReconcilePending = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				Type:  gardener.LastOperationTypeReconcile,
				State: gardener.LastOperationStatePending,
			},
		},
	}

	ShootLastOperationReconcileFailed = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				Type:  gardener.LastOperationTypeReconcile,
				State: gardener.LastOperationStateSucceeded,
			},
		},
	}

	ShootLastOperationReconcileSucceeded = gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: gardener.ShootSpec{
			DNS: &gardener.DNS{
				Domain: new(string),
			},
		},
		Status: gardener.ShootStatus{
			LastOperation: &gardener.LastOperation{
				Type:  gardener.LastOperationTypeReconcile,
				State: gardener.LastOperationStateSucceeded,
			},
		},
	}
)
