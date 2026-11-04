package evaluator

import (
	"testing"

	"github.com/imachange/devcontainer-parser/spec"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name string
		cfg  *spec.DevContainerConfig
		want ConfigType
	}{
		{"nil", nil, TypeUnknown},
		{"空", &spec.DevContainerConfig{}, TypeUnknown},
		{"image", &spec.DevContainerConfig{Image: "ubuntu"}, TypeImage},
		{"build", &spec.DevContainerConfig{Build: &spec.BuildSpec{Dockerfile: "Dockerfile"}}, TypeBuild},
		{"build (空のオブジェクト)", &spec.DevContainerConfig{Build: &spec.BuildSpec{}}, TypeBuild},
		{"旧形式の dockerFile", &spec.DevContainerConfig{DockerFile: "Dockerfile"}, TypeBuild},
		{"compose", &spec.DevContainerConfig{DockerComposeFile: spec.StringOrArray{"compose.yml"}}, TypeDockerCompose},
		{"compose (空の配列)", &spec.DevContainerConfig{DockerComposeFile: spec.StringOrArray{}}, TypeUnknown},
		// 優先順位: dockerComposeFile > build > image
		{"build と image", &spec.DevContainerConfig{Build: &spec.BuildSpec{}, Image: "ubuntu"}, TypeBuild},
		{"dockerFile と image", &spec.DevContainerConfig{DockerFile: "Dockerfile", Image: "ubuntu"}, TypeBuild},
		{"compose と build", &spec.DevContainerConfig{DockerComposeFile: spec.StringOrArray{"c.yml"}, Build: &spec.BuildSpec{}}, TypeDockerCompose},
		{"compose と image", &spec.DevContainerConfig{DockerComposeFile: spec.StringOrArray{"c.yml"}, Image: "ubuntu"}, TypeDockerCompose},
		{"3 つすべて", &spec.DevContainerConfig{DockerComposeFile: spec.StringOrArray{"c.yml"}, Build: &spec.BuildSpec{}, Image: "ubuntu"}, TypeDockerCompose},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Evaluate(tt.cfg); got != tt.want {
				t.Errorf("Evaluate = %q, want %q", got, tt.want)
			}
		})
	}
}
