package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v3"
)

type Store struct {
	path string
}

type contextKey struct{}

func WithPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, contextKey{}, path)
}

func PathFromContext(ctx context.Context) string {
	value, _ := ctx.Value(contextKey{}).(string)
	return value
}

func ResolvePath(override string) (string, error) {
	if override != "" {
		return filepath.Clean(override), nil
	}
	if value := os.Getenv("PRX_CONFIG"); value != "" {
		return filepath.Clean(value), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(dir, "prx", "config.yaml"), nil
}

func NewStore(path string) (*Store, error) {
	resolved, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}
	return &Store{path: resolved}, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Load() (Config, error) {
	var result Config
	err := s.withLock(false, func() error {
		body, err := os.ReadFile(s.path)
		if errors.Is(err, os.ErrNotExist) {
			result = Default()
			return nil
		}
		if err != nil {
			return fmt.Errorf("read config: %w", err)
		}
		if info, statErr := os.Stat(s.path); statErr == nil && info.Mode().Perm()&0o077 != 0 {
			return newError(ErrorCodeInvalid, "config file %q must have permissions 0600", s.path)
		}
		if err := decode(body, &result); err != nil {
			return err
		}
		result, err = result.Normalize()
		return err
	})
	if err != nil {
		return Config{}, err
	}
	return result, nil
}

func (s *Store) Save(value Config) error {
	normalized, err := value.Normalize()
	if err != nil {
		return err
	}
	return s.withLock(true, func() error {
		return s.saveLocked(normalized)
	})
}

func (s *Store) Update(update func(*Config) error) (Config, error) {
	var result Config
	err := s.withLock(true, func() error {
		var err error
		result, err = s.loadLocked()
		if err != nil {
			return err
		}
		if err := update(&result); err != nil {
			return err
		}
		result, err = result.Normalize()
		if err != nil {
			return err
		}
		return s.saveLocked(result)
	})
	if err != nil {
		return Config{}, err
	}
	return result, nil
}

func (s *Store) Public() (PublicConfig, error) {
	value, err := s.Load()
	if err != nil {
		return PublicConfig{}, err
	}
	return value.Public(), nil
}

func (s *Store) Validate() error {
	_, err := s.Load()
	return err
}

func decode(body []byte, destination *Config) error {
	decoder := yaml.NewDecoder(bytes.NewReader(body))
	decoder.KnownFields(true)
	if err := decoder.Decode(destination); err != nil {
		return newError(ErrorCodeInvalid, "decode config: %v", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return newError(ErrorCodeInvalid, "config must contain exactly one YAML document")
		}
		return newError(ErrorCodeInvalid, "decode config: %v", err)
	}
	return nil
}

func (s *Store) loadLocked() (Config, error) {
	body, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if info, statErr := os.Stat(s.path); statErr == nil && info.Mode().Perm()&0o077 != 0 {
		return Config{}, newError(ErrorCodeInvalid, "config file %q must have permissions 0600", s.path)
	}
	var result Config
	if err := decode(body, &result); err != nil {
		return Config{}, err
	}
	return result.Normalize()
}

func (s *Store) saveLocked(value Config) error {
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure config directory: %w", err)
	}
	var body bytes.Buffer
	encoder := yaml.NewEncoder(&body)
	encoder.SetIndent(2)
	if err := encoder.Encode(value); err != nil {
		_ = encoder.Close()
		return fmt.Errorf("encode config: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("close config encoder: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	temporaryName := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(temporaryName)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("secure temporary config: %w", err)
	}
	if _, err := temporary.Write(body.Bytes()); err != nil {
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(temporaryName, s.path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	keep = true
	directoryFile, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open config directory: %w", err)
	}
	defer func() { _ = directoryFile.Close() }()
	if err := directoryFile.Sync(); err != nil {
		return fmt.Errorf("sync config directory: %w", err)
	}
	return nil
}

func (s *Store) withLock(exclusive bool, fn func() error) error {
	if s.path == "" {
		return errors.New("config path is empty")
	}
	lockPath := s.path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return fmt.Errorf("create config lock directory: %w", err)
	}
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open config lock: %w", err)
	}
	defer func() { _ = lockFile.Close() }()
	operation := syscall.LOCK_SH
	if exclusive {
		operation = syscall.LOCK_EX
	}
	if err := syscall.Flock(int(lockFile.Fd()), operation); err != nil {
		return fmt.Errorf("lock config: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) }()
	return fn()
}
