/*
Copyright 2020 Red Hat, Inc.

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
	operators "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePackageManifests implements PackageManifestInterface
type FakePackageManifests struct {
	Fake *FakeOperators
	ns   string
}

var packagemanifestsResource = schema.GroupVersionResource{Group: "operators.coreos.com", Version: "", Resource: "packagemanifests"}

var packagemanifestsKind = schema.GroupVersionKind{Group: "operators.coreos.com", Version: "", Kind: "PackageManifest"}

// Get takes name of the packageManifest, and returns the corresponding packageManifest object, and an error if there is any.
func (c *FakePackageManifests) Get(name string, options v1.GetOptions) (result *operators.PackageManifest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(packagemanifestsResource, c.ns, name), &operators.PackageManifest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*operators.PackageManifest), err
}

// List takes label and field selectors, and returns the list of PackageManifests that match those selectors.
func (c *FakePackageManifests) List(opts v1.ListOptions) (result *operators.PackageManifestList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(packagemanifestsResource, packagemanifestsKind, c.ns, opts), &operators.PackageManifestList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &operators.PackageManifestList{ListMeta: obj.(*operators.PackageManifestList).ListMeta}
	for _, item := range obj.(*operators.PackageManifestList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested packageManifests.
func (c *FakePackageManifests) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(packagemanifestsResource, c.ns, opts))

}

// Create takes the representation of a packageManifest and creates it.  Returns the server's representation of the packageManifest, and an error, if there is any.
func (c *FakePackageManifests) Create(packageManifest *operators.PackageManifest) (result *operators.PackageManifest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(packagemanifestsResource, c.ns, packageManifest), &operators.PackageManifest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*operators.PackageManifest), err
}

// Update takes the representation of a packageManifest and updates it. Returns the server's representation of the packageManifest, and an error, if there is any.
func (c *FakePackageManifests) Update(packageManifest *operators.PackageManifest) (result *operators.PackageManifest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(packagemanifestsResource, c.ns, packageManifest), &operators.PackageManifest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*operators.PackageManifest), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePackageManifests) UpdateStatus(packageManifest *operators.PackageManifest) (*operators.PackageManifest, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(packagemanifestsResource, "status", c.ns, packageManifest), &operators.PackageManifest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*operators.PackageManifest), err
}

// Delete takes name of the packageManifest and deletes it. Returns an error if one occurs.
func (c *FakePackageManifests) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(packagemanifestsResource, c.ns, name), &operators.PackageManifest{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePackageManifests) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(packagemanifestsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &operators.PackageManifestList{})
	return err
}

// Patch applies the patch and returns the patched packageManifest.
func (c *FakePackageManifests) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *operators.PackageManifest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(packagemanifestsResource, c.ns, name, pt, data, subresources...), &operators.PackageManifest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*operators.PackageManifest), err
}
