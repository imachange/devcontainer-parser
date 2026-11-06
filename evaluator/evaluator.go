// Package evaluator は、devcontainer.json の構成方式 (Dockerfile / イメージ / Docker Compose) を判定。
package evaluator

import "github.com/imachange/devcontainer-parser/spec"

// ConfigType は、コンテナの構成方式。
type ConfigType string

const (
	// TypeBuild は Dockerfile による構成。
	TypeBuild ConfigType = "build"
	// TypeImage は Docker イメージによる構成。
	TypeImage ConfigType = "image"
	// TypeDockerCompose は Docker Compose による構成。
	TypeDockerCompose ConfigType = "dockerCompose"
	// TypeUnknown は不明な構成。
	TypeUnknown ConfigType = "unknown"
)

// Evaluate は、次の優先順位で、構成方式を判定。どれにも当てはまらなければ、TypeUnknown 。
//
//  1. dockerComposeFile (TypeDockerCompose)
//  2. build、または旧形式の dockerFile (TypeBuild)
//  3. image (TypeImage)
func Evaluate(config *spec.DevContainerConfig) ConfigType {
	if config == nil {
		return TypeUnknown
	}
	switch {
	case len(config.DockerComposeFile) > 0:
		return TypeDockerCompose
	case config.Build != nil || config.DockerFile != "":
		return TypeBuild
	case config.Image != "":
		return TypeImage
	}
	return TypeUnknown
}
