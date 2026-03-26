package api

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type EngineService struct {
	path string
	mu   sync.Mutex
}

func NewEngineService(path string) *EngineService {
	return &EngineService{
		path: path,
	}
}

type EngineLevel string

const (
	EngineEasy   EngineLevel = "easy"
	EngineMedium EngineLevel = "medium"
	EngineHard   EngineLevel = "hard"
)

func (e *EngineService) BestMove(fen string, level EngineLevel) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	cmd := exec.Command(e.path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("engine stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("engine stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start stockfish: %w", err)
	}

	reader := bufio.NewScanner(stdout)
	writer := func(line string) error {
		_, err := stdin.Write([]byte(line + "\n"))
		return err
	}

	if err := writer("uci"); err != nil {
		return "", err
	}
	if err := writer("isready"); err != nil {
		return "", err
	}

	for reader.Scan() {
		line := strings.TrimSpace(reader.Text())
		if line == "readyok" {
			break
		}
	}

	switch level {
	case EngineEasy:
		if err := writer("setoption name Skill Level value 2"); err != nil {
			return "", err
		}
		if err := writer("setoption name UCI_LimitStrength value true"); err != nil {
			return "", err
		}
		if err := writer("setoption name UCI_Elo value 900"); err != nil {
			return "", err
		}
	case EngineHard:
		if err := writer("setoption name Skill Level value 12"); err != nil {
			return "", err
		}
		if err := writer("setoption name UCI_LimitStrength value true"); err != nil {
			return "", err
		}
		if err := writer("setoption name UCI_Elo value 1700"); err != nil {
			return "", err
		}
	default:
		if err := writer("setoption name Skill Level value 6"); err != nil {
			return "", err
		}
		if err := writer("setoption name UCI_LimitStrength value true"); err != nil {
			return "", err
		}
		if err := writer("setoption name UCI_Elo value 1300"); err != nil {
			return "", err
		}
	}

	if err := writer("ucinewgame"); err != nil {
		return "", err
	}
	if err := writer("position fen " + fen); err != nil {
		return "", err
	}

	movetime := "200"
	switch level {
	case EngineEasy:
		movetime = "80"
	case EngineHard:
		movetime = "350"
	}

	if err := writer("go movetime " + movetime); err != nil {
		return "", err
	}

	bestMove := ""
	for reader.Scan() {
		line := strings.TrimSpace(reader.Text())
		if strings.HasPrefix(line, "bestmove ") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				bestMove = parts[1]
			}
			break
		}
	}

	_ = writer("quit")
	_ = stdin.Close()
	_ = cmd.Wait()

	if bestMove == "" || bestMove == "(none)" {
		return "", fmt.Errorf("engine returned no move")
	}

	return bestMove, nil
}
