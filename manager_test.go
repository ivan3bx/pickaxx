package pickaxx

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// safeStringBuilder is a StringBuilder that can be written/read concurrently.
type safeStringBuilder struct {
	*strings.Builder
	lock sync.Mutex
}

func (w *safeStringBuilder) Write(b []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.Builder.Write(b)
}

func (w *safeStringBuilder) WriteString(str string) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.Builder.WriteString(str)
}

func (w *safeStringBuilder) String() string {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.Builder.String()
}

func TestNewServerManager(t *testing.T) {
	t.Run("initialized state", func(t *testing.T) {
		m := &ProcessManager{}

		assert.Equal(t, Unknown, m.CurrentState())
		assert.False(t, m.Active())
		assert.Empty(t, m.RecentActivity())

		assert.Error(t, m.Stop(), "expected error on newly initialized server")
		assert.Error(t, m.Submit("/list"), "expect error on newly initialized server")
	})

	t.Run("starting", func(t *testing.T) {
		var writer *safeStringBuilder
		var m *ProcessManager

		tests := []struct {
			name      string
			checkFunc func(t *testing.T)
		}{
			{
				name: "is running",
				checkFunc: func(t *testing.T) {
					assertAsync(t, func() bool { return m.CurrentState() == Running })
				},
			},
			{
				name: "has I/O configured",
				checkFunc: func(t *testing.T) {
					assertAsync(t, func() bool { return m.cmdIn != nil && m.cmdOut != nil })
				},
			},
			{
				name: "sends status to clients",
				checkFunc: func(t *testing.T) {
					assertAsync(t, func() bool { return strings.Contains(writer.String(), `{"status":"Running"}`) })
				},
			},
			{
				name: "sends commands to executable",
				checkFunc: func(t *testing.T) {
					m.Submit("unique-input-string-123")
					assertAsync(t, func() bool {
						return strings.Contains(writer.String(), `unique-input-string-123`)
					})
				},
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {

				// new process manager
				m = &ProcessManager{
					Command:    []string{"cat"}, // simple input/output executable
					WorkingDir: os.TempDir(),
				}

				// create channels to observe state transitions
				stopCh := m.registerObserver(Stopped)
				runningCh := m.registerObserver(Running)

				defer func() {
					if m.cmd != nil {
						// test process won't shut-down gracefully, so do it the hard way
						m.cmd.Process.Kill()
						m.Stop()
						<-stopCh
					}
					m.unregister(stopCh)
					m.unregister(runningCh)
				}()

				writer = &safeStringBuilder{Builder: &strings.Builder{}}

				// Start the server
				if !assert.NoError(t, m.Start(writer)) {
					return
				}

				<-runningCh

				// Run our test
				tc.checkFunc(t)
			})
		}
	})
}

func assertAsync(t *testing.T, testFunc func() bool, msgs ...string) {
	const (
		timeout = time.Millisecond * 300
		tick    = time.Millisecond * 2
	)
	assert.Eventually(t, testFunc, timeout, tick, msgs)
}
