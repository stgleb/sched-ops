package k8s

import (
	"fmt"
	"time"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CRDOps is an interface to perfrom k8s Customer Resource operations
type CRDOps interface {
	// CreateCRD creates the given custom resource
	// This API will be deprecated soon. Use RegisterCRD instead
	CreateCRD(resource CustomResource) error
	// RegisterCRD creates the given custom resource
	RegisterCRD(crd *apiextensionsv1beta1.CustomResourceDefinition) error
	// ValidateCRD checks if the given CRD is registered
	ValidateCRD(resource CustomResource, timeout, retryInterval time.Duration) error
	// DeleteCRD deletes the CRD for the given complete name (plural.group)
	DeleteCRD(fullName string) error
}

// CustomResource is for creating a Kubernetes TPR/CRD
type CustomResource struct {
	// Name of the custom resource
	Name string
	// ShortNames are short names for the resource.  It must be all lowercase.
	ShortNames []string
	// Plural of the custom resource in plural
	Plural string
	// Group the custom resource belongs to
	Group string
	// Version which should be defined in a const above
	Version string
	// Scope of the CRD. Namespaced or cluster
	Scope apiextensionsv1beta1.ResourceScope
	// Kind is the serialized interface of the resource.
	Kind string
}

// CRD APIs - BEGIN

func (k *k8sOps) CreateCRD(resource CustomResource) error {
	if err := k.initK8sClient(); err != nil {
		return err
	}

	crdName := fmt.Sprintf("%s.%s", resource.Plural, resource.Group)
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   resource.Group,
			Version: resource.Version,
			Scope:   resource.Scope,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Singular:   resource.Name,
				Plural:     resource.Plural,
				Kind:       resource.Kind,
				ShortNames: resource.ShortNames,
			},
		},
	}

	_, err := k.apiExtensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return err
	}

	return nil
}

func (k *k8sOps) RegisterCRD(crd *apiextensionsv1beta1.CustomResourceDefinition) error {
	if err := k.initK8sClient(); err != nil {
		return err
	}

	_, err := k.apiExtensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return err
	}
	return nil
}

func (k *k8sOps) ValidateCRD(resource CustomResource, timeout, retryInterval time.Duration) error {
	if err := k.initK8sClient(); err != nil {
		return err
	}

	crdName := fmt.Sprintf("%s.%s", resource.Plural, resource.Group)
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		crd, err := k.apiExtensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, meta_v1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, nil
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					return false, fmt.Errorf("name conflict: %v", cond.Reason)
				}
			}
		}
		return false, nil
	})
}

func (k *k8sOps) DeleteCRD(fullName string) error {
	if err := k.initK8sClient(); err != nil {
		return err
	}

	return k.apiExtensionClient.ApiextensionsV1beta1().
		CustomResourceDefinitions().
		Delete(fullName, &meta_v1.DeleteOptions{PropagationPolicy: &deleteForegroundPolicy})
}

// CRD APIs - END
