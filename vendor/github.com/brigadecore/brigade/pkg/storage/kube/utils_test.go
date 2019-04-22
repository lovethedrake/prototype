package kube

import (
	"testing"

	v1 "k8s.io/api/storage/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

func TestStorageClassNames(t *testing.T) {
	k, s := fakeStore()
	createFakeStorageClasses(k)

	scn, err := s.GetStorageClassNames()
	if err != nil {
		t.Fatal(err)
	}

	if len(scn) != 2 {
		t.Fatal("StorageClass count should be 2")
	}
}

func createFakeStorageClasses(client kubernetes.Interface) {
	client.Storage().StorageClasses().Create(&v1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "sc1"}})
	client.Storage().StorageClasses().Create(&v1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "sc2"}})
}
