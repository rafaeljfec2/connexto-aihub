package execution

import (
	"context"
	"net/http"
	"testing"
	"time"
)

type quotaNoticeError struct{ resetIn time.Duration }

func (e *quotaNoticeError) Error() string { return "quota notice" }

func (e *quotaNoticeError) IsUpstreamQuotaNotice() bool { return true }

func (e *quotaNoticeError) QuotaResetIn() time.Duration { return e.resetIn }

func TestExecuteClassifiesUpstreamQuotaNotice(t *testing.T) {
	resetIn := 54*time.Hour + 24*time.Minute + 58*time.Second
	manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		return nil, &quotaNoticeError{resetIn: resetIn}
	})

	result, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if result.Outcome != OutcomeFailed || result.Phase != PhaseOpening || result.Cause != CauseUpstreamQuota {
		t.Fatalf("result = %+v, want failed/opening/upstream_quota", result)
	}
	if result.QuotaResetInNS == nil {
		t.Fatalf("QuotaResetInNS = nil, want parsed reset window")
	}
	if got := time.Duration(*result.QuotaResetInNS); got != resetIn {
		t.Fatalf("QuotaResetInNS = %v, want %v", got, resetIn)
	}
}
