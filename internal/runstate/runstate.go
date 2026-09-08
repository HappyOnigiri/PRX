// Package runstate は稼働中の `prx serve` を 1 つのファイルで記録する。多重起動防止の
// flock と稼働発見の記録を同じ inode に載せることで、アドレスを書いた本人が今も生きて
// いることをロック 1 つで保証する。docs/design/daemon.md を参照。
package runstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// SchemaVersion は記録の形式。読み手は将来の形式を無視できるよう、内容と一緒に保存する。
const SchemaVersion = 1

// DirEnvironmentVariable は記録の置き場所を差し替える。テストの子プロセスを実ユーザーの
// 記録から隔離するために必要で、`prx debug` の環境変数一覧にも載る。
const DirEnvironmentVariable = "PRX_RUN_DIR"

// State は稼働中のサーバーについて、外側のプロセスが知る必要のある事実を運ぶ。
type State struct {
	SchemaVersion  int       `json:"schema_version"`
	PID            int       `json:"pid"`
	Address        string    `json:"address"`
	URL            string    `json:"url"`
	StartedAt      time.Time `json:"started_at"`
	Version        string    `json:"version"`
	Executable     string    `json:"executable"`
	DatabasePath   string    `json:"database_path"`
	LaunchdManaged bool      `json:"launchd_managed"`
}

// Status は記録の読み手が観測できる 3 つの状態。内容の有効性はロックだけが決めるので、
// 残されたままの内容を稼働の根拠にはしない。
type Status string

const (
	StatusNotRunning            Status = "not_running"
	StatusRunning               Status = "running"
	StatusRunningAddressUnknown Status = "running_address_unknown"
)

// Dir は記録を置くディレクトリを返す。
func Dir() (string, error) {
	if override := os.Getenv(DirEnvironmentVariable); override != "" {
		return filepath.Clean(override), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve run state directory: %w", err)
	}
	return filepath.Join(dir, "prx", "run"), nil
}

// Path は記録ファイルの位置を返す。
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "serve.json"), nil
}

// Lock は保持中の排他ロックと、その内容を書く先を表す。
type Lock struct {
	file *os.File
	path string
}

// Path は保持しているロックファイルの位置を返す。
func (l *Lock) Path() string { return l.path }

// acquireAttempts と acquireRetryInterval は排他ロックの再試行を決める。読み手が Read の
// 中で一瞬だけ取る共有ロックと衝突しただけで、別のサーバーが稼働中だと誤判定すると、
// serve は終了コード 0 で退いてしまい launchd も再起動しない。
const (
	acquireAttempts      = 5
	acquireRetryInterval = 20 * time.Millisecond
)

// Acquire は排他ロックを試みる。別のサーバーが保持していれば held=false を返し、
// 呼び出し側は 2 つ目を起動しない。
func Acquire() (lock *Lock, held bool, err error) {
	path, err := Path()
	if err != nil {
		return nil, false, err
	}
	file, err := openPrivate(path)
	if err != nil {
		return nil, false, err
	}
	for attempt := 1; ; attempt++ {
		lockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if lockErr == nil {
			lock := &Lock{file: file, path: path}
			// 異常終了が残した内容を捨てる。ロックが移った後も残っていると、Write まで
			// の間だけ読み手が死んだサーバーのアドレスと pid を有効なものとして読む。
			if err := lock.truncate(); err != nil {
				_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
				_ = file.Close()
				return nil, false, err
			}
			return lock, true, nil
		}
		if !errors.Is(lockErr, syscall.EWOULDBLOCK) {
			_ = file.Close()
			return nil, false, fmt.Errorf("lock run state: %w", lockErr)
		}
		if attempt == acquireAttempts {
			_ = file.Close()
			return nil, false, nil
		}
		time.Sleep(acquireRetryInterval)
	}
}

// Write は稼働情報を in-place で置き換える。rename は inode を差し替えて保持中の flock を
// 無効にし、読み手が LOCK_SH を取れて未稼働と誤判定するので使わない。設定ファイルが
// rename で置き換わるのとは正反対の扱いになる。
func (l *Lock) Write(state State) error {
	state.SchemaVersion = SchemaVersion
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode run state: %w", err)
	}
	if err := l.truncate(); err != nil {
		return err
	}
	if _, err := l.file.WriteAt(data, 0); err != nil {
		return fmt.Errorf("write run state: %w", err)
	}
	return l.file.Sync()
}

// Release は内容を空にしてロックを手放す。unlink はしない。unlink とほぼ同時に新しい
// インスタンスが同じパスを作ってロックしていると、それを消してしまう。
func (l *Lock) Release() error {
	truncateErr := l.truncate()
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	return errors.Join(truncateErr, l.file.Close())
}

func (l *Lock) truncate() error {
	if err := l.file.Truncate(0); err != nil {
		return fmt.Errorf("truncate run state: %w", err)
	}
	return nil
}

// Read は記録を読み、稼働中かどうかを判定する。LOCK_SH が取れたということは書いた本人が
// もう居ないことなので、残っている内容は stale として捨てる。
func Read() (Status, State, error) {
	path, err := Path()
	if err != nil {
		return StatusNotRunning, State{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return StatusNotRunning, State{}, nil
		}
		return StatusNotRunning, State{}, fmt.Errorf("open run state: %w", err)
	}
	defer func() { _ = file.Close() }()
	if lockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); lockErr == nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		return StatusNotRunning, State{}, nil
	} else if !errors.Is(lockErr, syscall.EWOULDBLOCK) {
		return StatusNotRunning, State{}, fmt.Errorf("probe run state: %w", lockErr)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return StatusRunningAddressUnknown, State{}, nil
	}
	var state State
	if len(data) == 0 || json.Unmarshal(data, &state) != nil || state.Address == "" {
		return StatusRunningAddressUnknown, State{}, nil
	}
	return StatusRunning, state, nil
}

// openPrivate は 0700 のディレクトリに 0600 のファイルを用意する。既存ファイルの緩い
// モードを引き継がないよう、作成の成否によらず Chmod を呼ぶ。
func openPrivate(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create run state directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open run state: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("restrict run state permissions: %w", err)
	}
	return file, nil
}
