package diagnostics

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Risk string

const (
	RiskInfo     Risk = "info"
	RiskLow      Risk = "low"
	RiskMedium   Risk = "medium"
	RiskHigh     Risk = "high"
	RiskCritical Risk = "critical"
)

type Environment struct {
	Hostname       string   `json:"hostname"`
	OS             string   `json:"os"`
	Architecture   string   `json:"architecture"`
	InitSystem     string   `json:"init_system"`
	ContainerTools []string `json:"container_tools,omitempty"`
	AvailableTools []string `json:"available_tools,omitempty"`
}

type Finding struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	Risk        Risk   `json:"risk"`
	AutoFixable bool   `json:"auto_fixable"`
}

type Report struct {
	ID          string      `json:"id"`
	CollectedAt time.Time   `json:"collected_at"`
	Environment Environment `json:"environment"`
	RiskScore   int         `json:"risk_score"`
	Findings    []Finding   `json:"findings"`
}

type Remediation struct {
	ID              string `json:"id"`
	FindingID       string `json:"finding_id"`
	Title           string `json:"title"`
	Risk            Risk   `json:"risk"`
	Preview         string `json:"preview"`
	RequiresConfirm bool   `json:"requires_confirmation"`
	Rollback        string `json:"rollback"`
	Executable      bool   `json:"executable"`
}

func Collect(ctx context.Context) (Report, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	environment := Environment{
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
	for _, tool := range []string{"systemctl", "docker", "podman", "ufw", "firewall-cmd", "ss", "journalctl"} {
		if _, err := exec.LookPath(tool); err == nil {
			environment.AvailableTools = append(environment.AvailableTools, tool)
		}
	}
	if contains(environment.AvailableTools, "systemctl") {
		environment.InitSystem = "systemd"
	} else {
		environment.InitSystem = "unknown"
	}
	for _, tool := range []string{"docker", "podman"} {
		if contains(environment.AvailableTools, tool) {
			environment.ContainerTools = append(environment.ContainerTools, tool)
		}
	}

	findings := make([]Finding, 0, 4)
	if environment.InitSystem == "unknown" {
		findings = append(findings, Finding{
			ID: "init-system", Category: "compatibility", Title: "未识别到 systemd",
			Detail: "将跳过 systemd 专用操作，需要使用进程或容器适配器。", Risk: RiskLow,
		})
	}
	if len(environment.AvailableTools) == 0 {
		findings = append(findings, Finding{
			ID: "missing-tools", Category: "compatibility", Title: "基础运维工具缺失",
			Detail: "未发现可用于诊断的标准命令，自动修复能力将受限。", Risk: RiskMedium,
		})
	}
	if runtime.GOOS == "linux" && !contains(environment.AvailableTools, "ufw") && !contains(environment.AvailableTools, "firewall-cmd") {
		findings = append(findings, Finding{
			ID: "firewall-unknown", Category: "security", Title: "防火墙状态无法确认",
			Detail: "未找到 ufw 或 firewalld 管理工具，不代表防火墙一定关闭。", Risk: RiskMedium,
		})
	}
	if ctx.Err() != nil {
		return Report{}, ctx.Err()
	}

	report := Report{
		ID:          fmt.Sprintf("diag-%d", time.Now().UnixNano()),
		CollectedAt: time.Now().UTC(),
		Environment: environment,
		RiskScore:   score(findings),
		Findings:    findings,
	}
	return report, nil
}

func BuildPlan(report Report) []Remediation {
	plan := make([]Remediation, 0, len(report.Findings))
	for _, finding := range report.Findings {
		remediation := Remediation{
			ID:              "plan-" + finding.ID,
			FindingID:       finding.ID,
			Title:           "检查并处理：" + finding.Title,
			Risk:            finding.Risk,
			Preview:         finding.Detail,
			RequiresConfirm: finding.Risk == RiskHigh || finding.Risk == RiskCritical,
			Rollback:        "本检查项没有自动修改；执行前需由对应适配器提供备份和回滚步骤。",
			Executable:      false,
		}
		plan = append(plan, remediation)
	}
	return plan
}

func score(findings []Finding) int {
	score := 0
	for _, finding := range findings {
		switch finding.Risk {
		case RiskLow:
			score += 10
		case RiskMedium:
			score += 25
		case RiskHigh:
			score += 50
		case RiskCritical:
			score += 80
		}
	}
	if score > 100 {
		return 100
	}
	return score
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}
