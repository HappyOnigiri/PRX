package cli

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

// stubListener は bind の成否だけを扱うテスト用の listener。実ポートを掴むテストは
// 並列 CI で 7331 や 7332 を奪い合うので書かない。
type stubListener struct{ address string }

func (l stubListener) Accept() (net.Conn, error) { return nil, errors.New("stub listener") }

func (l stubListener) Close() error { return nil }

func (l stubListener) Addr() net.Addr { return stubAddr(l.address) }

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }

func (a stubAddr) String() string { return string(a) }

func TestPortListenPlanFallsBackOnlyForAuto(t *testing.T) {
	if plan := portListenPlan(config.ServerPortAuto); plan.address != "127.0.0.1:7331" || !plan.fallback {
		t.Fatalf("auto plan=%+v", plan)
	}
	if plan := portListenPlan(config.ServerPort(7400)); plan.address != "127.0.0.1:7400" || plan.fallback {
		t.Fatalf("fixed plan=%+v", plan)
	}
}

func TestListenServeFallsBackOnlyForAddressInUse(t *testing.T) {
	for _, test := range []struct {
		name        string
		plan        listenPlan
		bindErr     error
		wantAttempt []string
		wantCode    domain.DomainErrorCode
		wantErr     error
	}{
		{
			name:        "auto binds the preferred port",
			plan:        listenPlan{address: "127.0.0.1:7331", fallback: true},
			wantAttempt: []string{"127.0.0.1:7331"},
		},
		{
			name:        "auto escapes a busy preferred port",
			plan:        listenPlan{address: "127.0.0.1:7331", fallback: true},
			bindErr:     syscall.EADDRINUSE,
			wantAttempt: []string{"127.0.0.1:7331", "127.0.0.1:0"},
		},
		{
			name:        "auto keeps a privileged port failure",
			plan:        listenPlan{address: "127.0.0.1:80", fallback: true},
			bindErr:     syscall.EACCES,
			wantAttempt: []string{"127.0.0.1:80"},
			wantErr:     syscall.EACCES,
		},
		{
			name:        "a fixed port does not move",
			plan:        listenPlan{address: "127.0.0.1:7400"},
			bindErr:     syscall.EADDRINUSE,
			wantAttempt: []string{"127.0.0.1:7400"},
			wantCode:    domain.DomainErrorCodeAddressInUse,
		},
		{
			name:        "an explicit address does not move",
			plan:        listenPlan{address: "0.0.0.0:7331"},
			bindErr:     syscall.EADDRINUSE,
			wantAttempt: []string{"0.0.0.0:7331"},
			wantCode:    domain.DomainErrorCodeAddressInUse,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			attempted := make([]string, 0, 2)
			listen := func(_ context.Context, network, address string) (net.Listener, error) {
				if network != "tcp" {
					t.Fatalf("network=%q", network)
				}
				attempted = append(attempted, address)
				if test.bindErr != nil && address == test.plan.address {
					return nil, &net.OpError{Op: "listen", Err: test.bindErr}
				}
				return stubListener{address: address}, nil
			}
			listener, err := listenServe(context.Background(), listen, test.plan)
			if len(attempted) != len(test.wantAttempt) {
				t.Fatalf("attempted=%v, want %v", attempted, test.wantAttempt)
			}
			for index, address := range test.wantAttempt {
				if attempted[index] != address {
					t.Fatalf("attempted=%v, want %v", attempted, test.wantAttempt)
				}
			}
			switch {
			case test.wantCode != "":
				if domain.ErrorCode(err) != test.wantCode {
					t.Fatalf("error=%v, want code %q", err, test.wantCode)
				}
			case test.wantErr != nil:
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error=%v, want %v", err, test.wantErr)
				}
				if domain.ErrorCode(err) == domain.DomainErrorCodeAddressInUse {
					t.Fatalf("error=%v was reported as address_in_use", err)
				}
			default:
				if err != nil || listener == nil {
					t.Fatalf("listener=%v err=%v", listener, err)
				}
			}
		})
	}
}
