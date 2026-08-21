package buildinfo

import (
	"runtime"
	"runtime/debug"
	"strings"
)

// 以下变量通过 -ldflags "-X" 在编译期注入，使构建制品具备不可变、可追溯的 provenance。
var (
	// Version 是应用版本；未注入时为 "dev"。
	Version = "dev"
	// Commit 是构建对应的 Git 提交 SHA；未注入时为 "unknown"。
	Commit = "unknown"
	// BuildTime 是构建时间（RFC3339）；未注入时为 "unknown"。
	BuildTime = "unknown"
	// Environment 是构建所处的部署环境；未注入时为空。
	Environment = ""
)

// Dependency 描述一个构建依赖。
type Dependency struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// Info 汇总构建 provenance，用于「发布制品、Tag、Commit、测试报告和部署记录」之间互相追溯。
type Info struct {
	Version      string       `json:"version"`
	Commit       string       `json:"commit"`
	BuildTime    string       `json:"build_time"`
	Environment  string       `json:"environment,omitempty"`
	GoVersion    string       `json:"go_version"`
	Module       string       `json:"module"`
	Dependencies []Dependency `json:"dependencies"`
}

// Result 返回当前二进制的构建 provenance。
func Result() Info {
	info := Info{
		Version:     Version,
		Commit:      Commit,
		BuildTime:   BuildTime,
		Environment: Environment,
		GoVersion:   runtime.Version(),
	}

	build, ok := debug.ReadBuildInfo()
	if !ok || build == nil {
		return info
	}
	if build.GoVersion != "" {
		info.GoVersion = build.GoVersion
	}
	info.Module = build.Main.Path
	info.Dependencies = make([]Dependency, 0, len(build.Deps))
	for _, dep := range build.Deps {
		if dep == nil || dep.Path == "" {
			continue
		}
		info.Dependencies = append(info.Dependencies, Dependency{
			Path:    dep.Path,
			Version: dep.Version,
		})
	}
	return info
}

// DefaultVersion 返回未注入时的版本值，供配置层在未显式设置 VERSION 时回退。
func DefaultVersion() string {
	if v := strings.TrimSpace(Version); v != "" {
		return v
	}
	return "dev"
}
