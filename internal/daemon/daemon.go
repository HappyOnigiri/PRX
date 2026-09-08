// Package daemon は LaunchAgent の登録状態と稼働中サーバーの記録を 1 つの観測にまとめる。
// `prx daemon` と `prx debug` が同じ観測を読むので、両者の報告が食い違わない。
package daemon

import (
	"errors"
	"os"

	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/launchd"
	"github.com/HappyOnigiri/PRX/internal/runstate"
)

// Status は常駐について外から観測できる事実。
type Status struct {
	Supported bool
	// Installed は plist がディスク上にあることを表す。plist の内容が現在の PRX と
	// 一致するかは PlistStatus が持つ。
	Installed   bool
	PlistStatus launchd.PlistStatus
	PlistPath   string
	LogPath     string
	// Running は稼働記録のロックが保持されていることを表す。内容の有効性はロックだけが
	// 決めるので、記録が残っているだけでは稼働と見なさない。
	Running bool
	// AddressUnknown は稼働中だが記録の内容がまだ書かれていない、または壊れていることを表す。
	AddressUnknown bool
	State          runstate.State
	BinaryMatches  bool
	// Error は観測自体に失敗した理由。他のセクションを消さないよう、失敗もここに載せる。
	Error string
}

// Inspect は launchd と稼働記録を 1 度だけ読む。
func Inspect(manager *launchd.Manager, version string) Status {
	result := Status{Supported: manager.Supported(), BinaryMatches: true}
	if !result.Supported {
		return result
	}
	if path, err := manager.PlistPath(); err == nil {
		result.PlistPath = path
	} else {
		result.Error = err.Error()
	}
	if path, err := manager.LogPath(); err == nil {
		result.LogPath = path
	}
	status, err := manager.PlistStatus()
	result.PlistStatus = status
	switch {
	case err == nil:
		result.Installed = true
	case errors.Is(err, os.ErrNotExist):
		// 未導入は問題ではない。`prx serve` を手で使う運用も正当である。
	default:
		result.Error = err.Error()
	}
	running, state, err := runstate.Read()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	switch running {
	case runstate.StatusRunning:
		result.Running = true
		result.State = state
		result.BinaryMatches = binaryMatches(state, version)
	case runstate.StatusRunningAddressUnknown:
		result.Running = true
		result.AddressUnknown = true
	case runstate.StatusNotRunning:
	}
	return result
}

// binaryMatches は稼働中サーバーが今のバイナリで動いているかを返す。記録された実行
// ファイルが起動より後に更新されていれば置き換えられている。stat できないことを
// 不一致の根拠にはしない。誤検出は解決策のない問題として報告されてしまう。
func binaryMatches(state runstate.State, version string) bool {
	if state.Version != version {
		return false
	}
	info, err := os.Stat(state.Executable)
	if err != nil {
		return true
	}
	return !info.ModTime().After(state.StartedAt)
}

// DebugInput は診断レポート向けの入力に写す。domain は launchd も runstate も import
// できないので、変換はここで行う。
func (s Status) DebugInput(demo bool) domain.DebugDaemonInput {
	return domain.DebugDaemonInput{
		Supported:     s.Supported,
		Installed:     s.Installed,
		PlistStatus:   string(s.PlistStatus),
		PlistPath:     s.PlistPath,
		LogPath:       s.LogPath,
		Running:       s.Running,
		Address:       s.State.Address,
		PID:           s.State.PID,
		Version:       s.State.Version,
		BinaryMatches: s.BinaryMatches,
		Demo:          demo,
		Error:         s.Error,
	}
}
