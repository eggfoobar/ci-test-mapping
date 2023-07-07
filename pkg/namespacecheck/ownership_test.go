package namespacecheck

import (
	"encoding/json"
	"fmt"
	v1 "github.com/openshift-eng/ci-test-mapping/pkg/api/types/v1"
	"github.com/openshift-eng/ci-test-mapping/pkg/config"
	"github.com/openshift-eng/ci-test-mapping/pkg/registry"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNoOverlap(t *testing.T) {
	defaultRegistry := registry.NewComponentRegistry()

	componentNameToNamespaces := map[string][]string{}
	namespacesToComponentNames := map[string][]string{}

	for _, component := range defaultRegistry.Components {
		// All types currently conform.  If any don't in the future, find a way to handle them.
		// I recommend not skipping them.  Probably force conformance
		componentValuePointer := reflect.ValueOf(component)
		componentValue := componentValuePointer.Elem()
		defaultComponentValuePointer := componentValue.FieldByName("Component")
		asInterface := defaultComponentValuePointer.Interface()

		defaultComponent := asInterface.(*config.Component)
		for _, matcher := range defaultComponent.Matchers {
			for _, namespace := range matcher.Namespaces {
				namespacesToComponentNames[namespace] = append(namespacesToComponentNames[namespace], defaultComponent.Name)
				componentNameToNamespaces[defaultComponent.Name] = append(componentNameToNamespaces[defaultComponent.Name], namespace)
			}
		}

	}

	for namespace, components := range namespacesToComponentNames {
		if len(components) > 1 {
			t.Fatalf("namespace/%v is claimed by more than one component: %v", namespace, strings.Join(components, ", "))
		}
	}

	if !reflect.DeepEqual(componentNameToNamespaces, JiraComponentsToNamespaces) {
		// output something convenient to copy/paste
		for _, component := range sets.StringKeySet(componentNameToNamespaces).List() {
			fmt.Printf("%q: {\n", component)
			namespaces := componentNameToNamespaces[component]
			for _, namespace := range namespaces {
				fmt.Printf("    %q,\n", namespace)
			}
			fmt.Printf("},\n")
		}
		t.Errorf("mismatch between reuseable namespace mapping and perceived ownership")
	}

	namespacesToJiraComponent := map[string]string{}
	for k, v := range namespacesToComponentNames {
		namespacesToJiraComponent[k] = v[0]
	}

	if !reflect.DeepEqual(namespacesToJiraComponent, NamespacesToJiraComponents) {
		// output something convenient to copy/paste
		for _, component := range sets.StringKeySet(namespacesToJiraComponent).List() {
			namespace := namespacesToJiraComponent[component]
			fmt.Printf("%q: %q,\n", component, namespace)
		}
		t.Errorf("mismatch between reuseable namespace mapping and perceived ownership")
	}
}

func TestListOfKnownNamespaces(t *testing.T) {
	content, err := os.ReadFile("../../bigquery_tests.json")
	if err != nil {
		t.Fatal(err)
	}
	allTests := []v1.TestInfo{}
	if err := json.Unmarshal(content, &allTests); err != nil {
		t.Fatal(err)
	}

	foundNamespaces := sets.NewString()
	for _, test := range allTests {
		namespace := config.ExtractNamespaceFromTestName(test.Name)
		if len(namespace) > 0 {
			foundNamespaces.Insert(namespace)
		}
	}

	if !foundNamespaces.Equal(AllKnownNamespaces) {
		for _, ns := range foundNamespaces.List() {
			fmt.Printf("%q,\n", ns)
		}
		t.Error("mismatch in namespaces")
	}
}

func TestAllNamespacesAssignedWithoutExtras(t *testing.T) {
	assignedNamespaces := sets.StringKeySet(NamespacesToJiraComponents)
	if !assignedNamespaces.Equal(AllKnownNamespaces) {
		t.Log(assignedNamespaces.Difference(AllKnownNamespaces))
		t.Log(AllKnownNamespaces.Difference(assignedNamespaces))
		t.Errorf("not all namespaces are assigned")
	}
}
