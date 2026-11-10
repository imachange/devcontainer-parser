// Package spec は、devcontainer.json のデータモデルと、記法の揺れを吸収する型を提供する。
//
// モデルは、次のスキーマに対応。
//   - devcontainers/spec: schemas/devContainer.base.schema.json
//   - microsoft/vscode: extensions/configuration-editing/schemas/devContainer.codespaces.schema.json
//   - microsoft/vscode: extensions/configuration-editing/schemas/devContainer.vscode.schema.json
package spec

import (
	"encoding/json"
)

// DevContainerConfig は、devcontainer.json の、トップレベルのすべてのキー (TopLevelKeys) に対応。
type DevContainerConfig struct {
	// ---- 共通 (devContainerCommon)

	Schema                      string                     `json:"$schema,omitzero"`
	Name                        string                     `json:"name,omitzero"`
	Features                    map[string]json.RawMessage `json:"features,omitzero"`
	OverrideFeatureInstallOrder []string                   `json:"overrideFeatureInstallOrder,omitzero"`
	Secrets                     map[string]Secret          `json:"secrets,omitzero"`
	ForwardPorts                []PortMapping              `json:"forwardPorts,omitzero"`
	PortsAttributes             map[string]PortAttributes  `json:"portsAttributes,omitzero"`
	OtherPortsAttributes        *PortAttributes            `json:"otherPortsAttributes,omitzero"`
	UpdateRemoteUserUID         *bool                      `json:"updateRemoteUserUID,omitzero"`
	ContainerEnv                map[string]string          `json:"containerEnv,omitzero"`
	ContainerUser               string                     `json:"containerUser,omitzero"`
	Mounts                      []Mount                    `json:"mounts,omitzero"`
	Init                        *bool                      `json:"init,omitzero"`
	Privileged                  *bool                      `json:"privileged,omitzero"`
	CapAdd                      []string                   `json:"capAdd,omitzero"`
	SecurityOpt                 []string                   `json:"securityOpt,omitzero"`
	// RemoteEnv の値は、null を許す (その環境変数を取り除く)。
	RemoteEnv            map[string]*string `json:"remoteEnv,omitzero"`
	RemoteUser           string             `json:"remoteUser,omitzero"`
	InitializeCommand    LifecycleCommand   `json:"initializeCommand,omitzero"`
	OnCreateCommand      LifecycleCommand   `json:"onCreateCommand,omitzero"`
	UpdateContentCommand LifecycleCommand   `json:"updateContentCommand,omitzero"`
	PostCreateCommand    LifecycleCommand   `json:"postCreateCommand,omitzero"`
	PostStartCommand     LifecycleCommand   `json:"postStartCommand,omitzero"`
	PostAttachCommand    LifecycleCommand   `json:"postAttachCommand,omitzero"`
	WaitFor              string             `json:"waitFor,omitzero"`
	UserEnvProbe         string             `json:"userEnvProbe,omitzero"`
	HostRequirements     *HostRequirements  `json:"hostRequirements,omitzero"`
	// Customizations は、ツール固有の設定。ツールごとのキーの下は、生の JSON のまま保持する。
	Customizations map[string]json.RawMessage `json:"customizations,omitzero"`

	// ---- コンテナの起動方式が compose ではないときのキー (nonComposeBase。一部は compose と共通)

	AppPort         PortList `json:"appPort,omitzero"`
	RunArgs         []string `json:"runArgs,omitzero"`
	ShutdownAction  string   `json:"shutdownAction,omitzero"`
	OverrideCommand *bool    `json:"overrideCommand,omitzero"`
	WorkspaceFolder string   `json:"workspaceFolder,omitzero"`
	WorkspaceMount  string   `json:"workspaceMount,omitzero"`

	// ---- Dockerfile 方式 (dockerfileContainer)

	Build *BuildSpec `json:"build,omitzero"`
	// DockerFile と Context は、旧形式 (トップレベルの dockerFile / context)。
	DockerFile string `json:"dockerFile,omitzero"`
	Context    string `json:"context,omitzero"`

	// ---- イメージ方式 (imageContainer)

	Image string `json:"image,omitzero"`

	// ---- Docker Compose 方式 (composeContainer)

	DockerComposeFile StringOrArray `json:"dockerComposeFile,omitzero"`
	Service           string        `json:"service,omitzero"`
	RunServices       []string      `json:"runServices,omitzero"`

	// ---- VS Code / Codespaces 用のスキーマ (どちらも、トップレベルの旧形式は非推奨)

	Extensions []string        `json:"extensions,omitzero"`
	Settings   json.RawMessage `json:"settings,omitzero"`
	DevPort    *int            `json:"devPort,omitzero"`
	Codespaces json.RawMessage `json:"codespaces,omitzero"`
}

// Secret は、secrets の値。
type Secret struct {
	Description      string `json:"description,omitzero"`
	DocumentationURL string `json:"documentationUrl,omitzero"`
}

// PortAttributes は、portsAttributes / otherPortsAttributes の値。
type PortAttributes struct {
	OnAutoForward    string `json:"onAutoForward,omitzero"`
	ElevateIfNeeded  *bool  `json:"elevateIfNeeded,omitzero"`
	Label            string `json:"label,omitzero"`
	RequireLocalPort *bool  `json:"requireLocalPort,omitzero"`
	Protocol         string `json:"protocol,omitzero"`
}

// HostRequirements は、hostRequirements 。
type HostRequirements struct {
	CPUs    *int            `json:"cpus,omitzero"`
	Memory  string          `json:"memory,omitzero"`
	Storage string          `json:"storage,omitzero"`
	GPU     *GPURequirement `json:"gpu,omitzero"`
}

// Decode は、トップレベルのキーと値から、DevContainerConfig を作る。
//
// encoding/json は、キーの大文字小文字を区別せずに照合するので、そのまま読むと、仕様に無いキー
// ("dockerfile" など) を、仕様のキー ("dockerFile") として読んでしまう。
// それを避けるため、仕様のキーと完全に一致するものだけを残してから、読む。
// (入れ子の中のキーは、encoding/json の照合のまま。)
func Decode(top map[string]json.RawMessage) (*DevContainerConfig, error) {
	known := make(map[string]json.RawMessage, len(top))
	for k, v := range top {
		if IsKnownKey(k) {
			known[k] = v
		}
	}
	b, err := json.Marshal(known)
	if err != nil {
		return nil, err
	}
	var cfg DevContainerConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
