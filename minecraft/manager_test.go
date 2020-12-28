package minecraft

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ivan3bx/pickaxx"
	"github.com/stretchr/testify/assert"
)

var dataDir string

func init() {
	dataDir, _ = ioutil.TempDir(os.TempDir(), "pickaxx_minecraft")
	log.WithField("path", dataDir).Info("test directory")

	log.SetLevel(log.ErrorLevel)
}
func TestNewServerManager(t *testing.T) {
	t.Run("initialized state", func(t *testing.T) {
		m := New(dataDir, DefaultPort)

		assert.False(t, m.Running())
		assert.Error(t, m.Stop(), "expected error on newly initialized server")
		assert.Error(t, m.Submit("/list"), "expect error on newly initialized server")
	})

	t.Run("starting", func(t *testing.T) {
		var m *serverManager

		tests := []struct {
			name      string
			checkFunc func(t *testing.T, activity <-chan pickaxx.Data)
		}{
			{
				name: "is running",
				checkFunc: func(t *testing.T, activity <-chan pickaxx.Data) {
					assertAsync(t, func() bool { return m.state == Running })
				},
			},
			{
				name: "has I/O configured",
				checkFunc: func(t *testing.T, activity <-chan pickaxx.Data) {
					assertAsync(t, func() bool { return m.cmdIn != nil })
				},
			},
			{
				name: "sends status to clients",
				checkFunc: func(t *testing.T, activity <-chan pickaxx.Data) {
					assertAsync(t, func() bool {
						output, _ := json.Marshal(<-activity)
						return strings.Contains(string(output), `{"status":"Running"}`)
					})
				},
			},
			{
				name: "sends commands to executable",
				checkFunc: func(t *testing.T, activity <-chan pickaxx.Data) {
					m.Submit("unique-input-string-123")
					assertAsync(t, func() bool {
						output, _ := json.Marshal(<-activity)
						return strings.Contains(string(output), `unique-input-string-123`)
					})
				},
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {

				// new process manager
				m = &serverManager{
					Command:    []string{"cat"}, // simple input/output executable
					WorkingDir: os.TempDir(),
				}

				// create channels to observe state transitions
				isRunning := m.notifier.Register(Running)

				defer func() {
					if m.cmd != nil {
						m.cmd.Process.Kill() // test process must be killed
						m.Stop()
					}
					m.notifier.Unregister(isRunning)
				}()

				// Start the server
				activity, err := m.Start()

				if !assert.NoError(t, err) {
					return
				}

				<-isRunning

				// Run our test
				tc.checkFunc(t, activity)
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
