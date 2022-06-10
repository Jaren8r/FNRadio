package logreader

import (
	"io"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type LogReader struct {
	file     *os.File
	fileName string
	watcher  *fsnotify.Watcher
	Events   chan LogEvent
}

type LogEvent struct {
	Initial bool
	Lines   []string
	Error   error
}

func New(file string) (*LogReader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	reader := &LogReader{
		fileName: file,
		watcher:  watcher,
		Events:   make(chan LogEvent),
	}

	go reader.start()

	return reader, nil
}

func (reader *LogReader) handleLines(initial bool, rawLines string) {
	split := strings.Split(rawLines, "\r\n")
	lines := make([]string, 0)

	for _, line := range split {
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}

	reader.Events <- LogEvent{
		Initial: initial,
		Lines:   lines,
	}
}

func (reader *LogReader) handleEvents() {
	for event := range reader.watcher.Events {
		if event.Op == fsnotify.Write {
			stat, err := reader.file.Stat()
			if err != nil {
				return
			}

			offset, err := reader.file.Seek(0, io.SeekCurrent)
			if err != nil {
				return
			}

			if stat.Size() < offset {
				_, err := reader.file.Seek(0, io.SeekStart)
				if err != nil {
					return
				}
			}

			file, err := io.ReadAll(reader.file)
			if err != nil {
				return
			}

			reader.handleLines(false, string(file))
		}
	}
}

func (reader *LogReader) start() {
	var err error

	reader.file, err = os.Open(reader.fileName)
	if err != nil {
		reader.Events <- LogEvent{
			Error: err,
		}
		close(reader.Events)

		return
	}

	err = reader.watcher.Add(reader.fileName)
	if err != nil {
		reader.Events <- LogEvent{
			Error: err,
		}
		close(reader.Events)

		return
	}

	initial, err := io.ReadAll(reader.file)
	if err != nil {
		reader.Events <- LogEvent{
			Error: err,
		}
		close(reader.Events)

		return
	}

	reader.handleLines(true, string(initial))

	reader.handleEvents()
}
