package app

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func (a *App) runVfoxWithProgressLocked(args []string) error {
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return err
	}
	defer releaseTask()
	return a.runVfoxWithProgress(args)
}

func (a *App) runVfoxWithProgress(args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd, err := a.newVfoxProgressCommand(ctx, args)
	if err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		a.emitEvent("vfox-log", fmt.Sprintf("[EXIT ERROR] StdoutPipe failed: %v", err))
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		a.emitEvent("vfox-log", fmt.Sprintf("[EXIT ERROR] StderrPipe failed: %v", err))
		return err
	}

	if err := cmd.Start(); err != nil {
		a.emitEvent("vfox-log", fmt.Sprintf("[EXIT ERROR] cmd.Start failed: %v", err))
		return err
	}

	outputState := &vfoxProgressOutputState{}

	var readWG sync.WaitGroup
	readWG.Add(2)
	go a.readVfoxProgressPipe(stdoutPipe, "STDOUT", outputState, &readWG)
	go a.readVfoxProgressPipe(stderrPipe, "STDERR", outputState, &readWG)

	err = cmd.Wait()
	readWG.Wait()
	return a.finishVfoxProgressCommand(ctx, args, outputState, err)
}

func (a *App) newVfoxProgressCommand(ctx context.Context, args []string) (*exec.Cmd, error) {
	vfoxExe, err := a.getVfoxExecutable()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, vfoxExe, args...)
	hideWindow(cmd)
	cmd.Stdin = strings.NewReader("y\ny\ny\ny\ny\n")
	cmd.Env = a.getCleanedEnvForVfox()
	return cmd, nil
}

const versionNotReleasedErrorMessage = "version is not released"

func isVersionNotReleasedOutput(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, versionNotReleasedErrorMessage)
}

type vfoxProgressOutputState struct {
	mu                  sync.Mutex
	lastOutput          string
	releaseStatusOutput string
}

func (state *vfoxProgressOutputState) record(line string) {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.lastOutput = line
	if isVersionNotReleasedOutput(line) {
		state.releaseStatusOutput = line
	}
}

func (state *vfoxProgressOutputState) errorDetail(fallback error) string {
	state.mu.Lock()
	defer state.mu.Unlock()

	detail := strings.TrimSpace(state.lastOutput)
	if state.releaseStatusOutput != "" {
		return versionNotReleasedErrorMessage
	}
	if detail == "" {
		return fallback.Error()
	}
	return fmt.Sprintf("%s (%v)", detail, fallback)
}

func splitVfoxProgressLine(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func (a *App) readVfoxProgressPipe(pipe io.Reader, label string, outputState *vfoxProgressOutputState, readWG *sync.WaitGroup) {
	defer readWG.Done()

	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	scanner.Split(splitVfoxProgressLine)
	for scanner.Scan() {
		if a.ctx == nil {
			continue
		}
		a.emitVfoxProgressOutput(scanner.Text(), outputState)
	}
	if err := scanner.Err(); err != nil && a.ctx != nil {
		a.emitEvent("vfox-log", fmt.Sprintf("[%s READ ERROR] %s", label, err.Error()))
	}
}

func (a *App) emitVfoxProgressOutput(line string, outputState *vfoxProgressOutputState) {
	cleanLine := ansiRegex.ReplaceAllString(line, "")
	if cleanLine == "" {
		return
	}
	outputState.record(cleanLine)
	a.emitEvent("vfox-log", cleanLine)
}

func (a *App) finishVfoxProgressCommand(ctx context.Context, args []string, outputState *vfoxProgressOutputState, err error) error {
	if ctx.Err() == context.DeadlineExceeded {
		a.emitEvent("vfox-log", "[TIMEOUT] Command cancelled after 30min")
		if err != nil {
			return fmt.Errorf("vfox %v timed out after 30min: %w", args, err)
		}
		return fmt.Errorf("vfox %v timed out after 30min", args)
	}
	if err != nil {
		detail := outputState.errorDetail(err)
		a.emitEvent("vfox-log", "[EXIT ERROR] "+detail)
		return fmt.Errorf("%s", detail)
	}

	a.emitEvent("vfox-log", "[DONE]")
	return nil
}
