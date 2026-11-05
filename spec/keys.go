package spec

import "sort"

// TopLevelKeys は、スキーマが定義する、devcontainer.json のトップレベルのキーの一覧 (辞書順)。
//
// 出典:
//   - base: devContainerCommon、nonComposeBase、dockerfileContainer、imageContainer、composeContainer
//   - vscode: extensions、settings、devPort (どれも非推奨。customizations.vscode を使う)
//   - codespaces: codespaces (非推奨。customizations.codespaces を使う)
//   - vscode / codespaces: customizations (base と同じキー)
//
var TopLevelKeys = []string{
	"$schema",
	"appPort",
	"build",
	"capAdd",
	"codespaces",
	"containerEnv",
	"containerUser",
	"context",
	"customizations",
	"devPort",
	"dockerComposeFile",
	"dockerFile",
	"extensions",
	"features",
	"forwardPorts",
	"hostRequirements",
	"image",
	"init",
	"initializeCommand",
	"mounts",
	"name",
	"onCreateCommand",
	"otherPortsAttributes",
	"overrideCommand",
	"overrideFeatureInstallOrder",
	"portsAttributes",
	"postAttachCommand",
	"postCreateCommand",
	"postStartCommand",
	"privileged",
	"remoteEnv",
	"remoteUser",
	"runArgs",
	"runServices",
	"secrets",
	"securityOpt",
	"service",
	"settings",
	"shutdownAction",
	"updateContentCommand",
	"updateRemoteUserUID",
	"userEnvProbe",
	"waitFor",
	"workspaceFolder",
	"workspaceMount",
}

var knownKeys = func() map[string]struct{} {
	m := make(map[string]struct{}, len(TopLevelKeys))
	for _, k := range TopLevelKeys {
		m[k] = struct{}{}
	}
	return m
}()

// IsKnownKey は、スキーマが定義するトップレベルのキーかを返す (大文字小文字を区別する)。
func IsKnownKey(key string) bool {
	_, ok := knownKeys[key]
	return ok
}

// SortedKeys は、マップのキーを、辞書順で返す。
func SortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
