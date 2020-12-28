package minecraft

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ivan3bx/pickaxx"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetLevel(log.ErrorLevel)
}
func TestNewServerManager(t *testing.T) {
	t.Run("initialized state", func(t *testing.T) {
		m := New(DefaultPort)

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
				w := dummyMonitor{isReady: make(chan bool)}
				err := m.Start(&w)

				if !assert.NoError(t, err) {
					return
				}

				<-isRunning
				<-w.isReady

				// Run our test
				tc.checkFunc(t, w.activityCh)
			})
		}
	})
}

type dummyMonitor struct {
	isReady    chan bool // signals when channel can retrieve data
	activityCh <-chan pickaxx.Data
}

func (c *dummyMonitor) Accept(ch <-chan pickaxx.Data) error {
	c.activityCh = ch // set the activity channel
	c.isReady <- true // signal that monitor is wired up
	return nil
}

func assertAsync(t *testing.T, testFunc func() bool, msgs ...string) {
	const (
		timeout = time.Millisecond * 300
		tick    = time.Millisecond * 2
	)
	assert.Eventually(t, testFunc, timeout, tick, msgs)
}
