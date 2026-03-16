package k8s

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type listResourcesOutput struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
	Status    string `yaml:"status,omitempty"`
	Ready     string `yaml:"ready,omitempty"`
	Created   string `yaml:"created,omitempty"`
}

type apiResourcesOutput struct {
	Name       string `yaml:"name"`
	Kind       string `yaml:"kind"`
	Group      string `yaml:"group"`
	Namespaced bool   `yaml:"namespaced"`
}

type eventOutput struct {
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
	Kind      string   `yaml:"kind"`
	Reason    string   `yaml:"reason"`
	Message   string   `yaml:"message"`
	Type      string   `yaml:"type"`
	Count     int32    `yaml:"count"`
	FirstTime yamlTime `yaml:"firstTime"`
	LastTime  yamlTime `yaml:"lastTime"`
}

type nodeTopOutput struct {
	Name     string `yaml:"name"`
	CPU      string `yaml:"cpu"`
	Memory   string `yaml:"memory"`
	Storage  string `yaml:"storage"`
	StorageE string `yaml:"storageE"`
}

type podTopOutput struct {
	Name       string               `yaml:"name"`
	Namespace  string               `yaml:"namespace"`
	CPU        string               `yaml:"cpu"`
	Memory     string               `yaml:"memory"`
	Containers []containerTopOutput `yaml:"containers"`
}

type containerTopOutput struct {
	Name   string `yaml:"name"`
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type yamlTime struct {
	metav1.MicroTime
}

func (t yamlTime) MarshalYAML() (interface{}, error) {
	if t.IsZero() {
		return "", nil
	}
	return t.UTC().Format(time.DateTime), nil
}
