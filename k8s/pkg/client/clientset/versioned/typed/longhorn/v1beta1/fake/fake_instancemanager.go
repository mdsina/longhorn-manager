/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1beta1 "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeInstanceManagers implements InstanceManagerInterface
type FakeInstanceManagers struct {
	Fake *FakeLonghornV1beta1
	ns   string
}

var instancemanagersResource = schema.GroupVersionResource{Group: "longhorn.io", Version: "v1beta1", Resource: "instancemanagers"}

var instancemanagersKind = schema.GroupVersionKind{Group: "longhorn.io", Version: "v1beta1", Kind: "InstanceManager"}

// Get takes name of the instanceManager, and returns the corresponding instanceManager object, and an error if there is any.
func (c *FakeInstanceManagers) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.InstanceManager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(instancemanagersResource, c.ns, name), &v1beta1.InstanceManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.InstanceManager), err
}

// List takes label and field selectors, and returns the list of InstanceManagers that match those selectors.
func (c *FakeInstanceManagers) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.InstanceManagerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(instancemanagersResource, instancemanagersKind, c.ns, opts), &v1beta1.InstanceManagerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.InstanceManagerList{ListMeta: obj.(*v1beta1.InstanceManagerList).ListMeta}
	for _, item := range obj.(*v1beta1.InstanceManagerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested instanceManagers.
func (c *FakeInstanceManagers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(instancemanagersResource, c.ns, opts))

}

// Create takes the representation of a instanceManager and creates it.  Returns the server's representation of the instanceManager, and an error, if there is any.
func (c *FakeInstanceManagers) Create(ctx context.Context, instanceManager *v1beta1.InstanceManager, opts v1.CreateOptions) (result *v1beta1.InstanceManager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(instancemanagersResource, c.ns, instanceManager), &v1beta1.InstanceManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.InstanceManager), err
}

// Update takes the representation of a instanceManager and updates it. Returns the server's representation of the instanceManager, and an error, if there is any.
func (c *FakeInstanceManagers) Update(ctx context.Context, instanceManager *v1beta1.InstanceManager, opts v1.UpdateOptions) (result *v1beta1.InstanceManager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(instancemanagersResource, c.ns, instanceManager), &v1beta1.InstanceManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.InstanceManager), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeInstanceManagers) UpdateStatus(ctx context.Context, instanceManager *v1beta1.InstanceManager, opts v1.UpdateOptions) (*v1beta1.InstanceManager, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(instancemanagersResource, "status", c.ns, instanceManager), &v1beta1.InstanceManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.InstanceManager), err
}

// Delete takes name of the instanceManager and deletes it. Returns an error if one occurs.
func (c *FakeInstanceManagers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(instancemanagersResource, c.ns, name), &v1beta1.InstanceManager{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeInstanceManagers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(instancemanagersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.InstanceManagerList{})
	return err
}

// Patch applies the patch and returns the patched instanceManager.
func (c *FakeInstanceManagers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.InstanceManager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(instancemanagersResource, c.ns, name, pt, data, subresources...), &v1beta1.InstanceManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.InstanceManager), err
}
