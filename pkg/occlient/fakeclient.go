package occlient

import (
	fkrouteClientset "github.com/openshift/client-go/route/clientset/versioned/fake"
)

// fkClientSet holds fake ClientSets
// this is returned by FakeNew to access methods of fake client sets
type FkClientSet struct {
	RouteClientset *fkrouteClientset.Clientset
}

// FakeNew create new fake client for testing
// returns Client that is filled with fake clients and
// fkClientSet that holds fake Clientsets to access Actions, Reactors etc... in fake client
func FakeNew() (*Client, *FkClientSet) {
	var client Client
	var fkclientset FkClientSet

	fkclientset.RouteClientset = fkrouteClientset.NewSimpleClientset()
	client.routeClient = fkclientset.RouteClientset.Route()

	return &client, &fkclientset
}
