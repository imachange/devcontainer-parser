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

	Schema                      string                     `json:"$schema,omitempty"`
	Name                        string                     `json:"name,omitempty"`
	Features                    map[string]json.RawMessage `json:"features,omitempty"`
	OverrideFeatureInstallOrder []string                   `json:"overrideFeatureInstallOrder,omitempty"`
	Secrets                     map[string]Secret          `json:"secrets,omitempty"`
	ForwardPorts                []PortMapping              `json:"forwardPorts,omitempty"`
	PortsAttributes             map[string]PortAttributes  `json:"portsAttributes,omitempty"`
	OtherPortsAttributes        *PortAttributes            `json:"otherPortsAttributes,omitempty"`
	UpdateRemoteUserUID         *bool                      `json:"updateRemoteUserUID,omitempty"`
	ContainerEnv                map[string]string          `json:"containerEnv,omitempty"`
	ContainerUser               string                     `json:"containerUser,omitempty"`
	Mounts                      []Mount                    `json:"mounts,omitempty"`
	Init                        *bool                      `json:"init,omitempty"`
	Privileged                  *bool                      `json:"privileged,omitempty"`
	CapAdd                      []string                   `json:"capAdd,omitempty"`
	SecurityOpt                 []string                   `json:"securityOpt,omitempty"`
	// RemoteEnv の値は、null を許します (その環境変数を取り除く)。
	RemoteEnv            map[string]*string `json:"remoteEnv,omitempty"`
	RemoteUser           string             `json:"remoteUser,omitempty"`
	InitializeCommand    LifecycleCommand   `json:"initializeCommand,omitzero"`
	OnCreateCommand      LifecycleCommand   `json:"onCreateCommand,omitzero"`
	UpdateContentCommand LifecycleCommand   `json:"updateContentCommand,omitzero"`
	PostCreateCommand    LifecycleCommand   `json:"postCreateCommand,omitzero"`
	PostStartCommand     LifecycleCommand   `json:"postStartCommand,omitzero"`
	PostAttachCommand    LifecycleCommand   `json:"postAttachCommand,omitzero"`
	WaitFor              string             `json:"waitFor,omitempty"`
	UserEnvProbe         string             `json:"userEnvProbe,omitempty"`
	HostRequirements     *HostRequirements  `json:"hostRequirements,omitempty"`
	// Customizations は、ツール固有の設定です。ツールごとのキーの下は、生の JSON のまま保持する。
	Customizations map[string]json.RawMessage `json:"customizations,omitempty"`

	// ---- コンテナの起動方式が compose ではないときのキー (nonComposeBase。一部は compose と共通)

	AppPort         PortList `json:"appPort,omitempty"`
	RunArgs         []string `json:"runArgs,omitempty"`
	ShutdownAction  string   `json:"shutdownAction,omitempty"`
	OverrideCommand *bool    `json:"overrideCommand,omitempty"`
	WorkspaceFolder string   `json:"workspaceFolder,omitempty"`
	WorkspaceMount  string   `json:"workspaceMount,omitempty"`

	// ---- Dockerfile 方式 (dockerfileContainer)

	Build *BuildSpec `json:"build,omitempty"`
	// DockerFile と Context は、旧形式 (トップレベルの dockerFile / context) です。
	DockerFile string `json:"dockerFile,omitempty"`
	Context    string `json:"context,omitempty"`

	// ---- イメージ方式 (imageContainer)

	Image string `json:"image,omitempty"`

	// ---- Docker Compose 方式 (composeContainer)

	DockerComposeFile StringOrArray `json:"dockerComposeFile,omitempty"`
	Service           string        `json:"service,omitempty"`
	RunServices       []string      `json:"runServices,omitempty"`

	// ---- VS Code / Codespaces 用のスキーマ (どちらも、トップレベルの旧形式は非推奨)

	Extensions []string        `json:"extensions,omitempty"`
	Settings   json.RawMessage `json:"settings,omitempty"`
	DevPort    *int            `json:"devPort,omitempty"`
	Codespaces json.RawMessage `json:"codespaces,omitempty"`
}

// Secret は、secrets の値。
type Secret struct {
	Description      string `json:"description,omitempty"`
	DocumentationURL string `json:"documentationUrl,omitempty"`
}

// PortAttributes は、portsAttributes / otherPortsAttributes の値。
type PortAttributes struct {
	OnAutoForward    string `json:"onAutoForward,omitempty"`
	ElevateIfNeeded  *bool  `json:"elevateIfNeeded,omitempty"`
	Label            string `json:"label,omitempty"`
	RequireLocalPort *bool  `json:"requireLocalPort,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
}

// HostRequirements は、hostRequirements 。
type HostRequirements struct {
	CPUs    *int            `json:"cpus,omitempty"`
	Memory  string          `json:"memory,omitempty"`
	Storage string          `json:"storage,omitempty"`
	GPU     *GPURequirement `json:"gpu,omitempty"`
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
