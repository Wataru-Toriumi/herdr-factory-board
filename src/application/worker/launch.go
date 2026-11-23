package worker

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/Wataru-Toriumi/herdr-factory-board/src/domain/board"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/herdr"
	"github.com/Wataru-Toriumi/herdr-factory-board/src/infrastructure/store"
)

// Herdr が起動するワーカーペインに、対象カードと実行番号を一組で渡す。
const (
	cardIDEnv  = "FACTORY_CARD"
	attemptEnv = "FACTORY_ATTEMPT"
)

func Start(s store.Store, herdrBin string, id int) error {
	return launchCard(s, herdrBin, id, func(c *board.Card) error {
		if c.Status != board.Todo && c.Status != board.Failed {
			return fmt.Errorf("#%d は %s です", id, c.Status)
		}
		return nil
	})
}

func launchCard(s store.Store, herdrBin string, id int, prepare func(*board.Card) error) error {
	var attempt int
	err := s.EditCard(id, func(c *board.Card) error {
		if err := prepare(c); err != nil {
			return err
		}
		attempt = c.QueueAttempt()
		return nil
	})
	if err != nil {
		return err
	}
	launchErr := herdr.OpenPlugin(herdrBin, "worker", cardIDEnv+"="+strconv.Itoa(id), attemptEnv+"="+strconv.Itoa(attempt))
	if launchErr == nil {
		return nil
	}
	saveErr := s.EditCard(id, func(c *board.Card) error {
		c.FailAttemptLaunch(attempt, launchErr.Error())
		return nil
	})
	return errors.Join(launchErr, saveErr)
}

func readLaunchRequest() (cardID, attempt int, err error) {
	cardID, err = strconv.Atoi(os.Getenv(cardIDEnv))
	if err != nil {
		return 0, 0, errors.New("ボードから実行してください")
	}
	attempt, err = strconv.Atoi(os.Getenv(attemptEnv))
	return cardID, attempt, err
}
