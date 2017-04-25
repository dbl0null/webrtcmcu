package llog

import "log"
import "fmt"
import "bytes"
import "time"
import "io"
import "os"

const (
	DEBUG = 1
	INFO  = 2
	WARN  = 3
	ERROR = 4
	FETAL = 5
)

type Logger struct {
	l     *log.Logger
	level int
}

func (log *Logger) log(level int, format string, vars ...interface{}) {
	if level >= log.level {
		log.l.Output(3, fmt.Sprintf(format, vars...))
	}
}
func (log *Logger) Debug(format string, vars ...interface{}) {
	log.log(DEBUG, format, vars...)
}

func (log *Logger) Info(format string, vars ...interface{}) {
	log.log(INFO, format, vars...)
}

func (log *Logger) Warn(format string, vars ...interface{}) {
	log.log(WARN, format, vars...)
}

func (log *Logger) Error(format string, vars ...interface{}) {
	log.log(ERROR, format, vars...)
}

func (log *Logger) Fetal(format string, vars ...interface{}) {
	log.log(FETAL, format, vars...)
}

type AsyncBuffWriter struct {
	c       chan []byte
	bufsize int
	w       io.Writer
}

func (w *AsyncBuffWriter) Write(b []byte) (int, error) {
	w.c <- b
	return len(b), nil
}

func (w *AsyncBuffWriter) run() {
	go func() {
		buf := bytes.Buffer{}
		for {
			select {
			case b := <-w.c:
				buf.Write(b)
				if buf.Len() > w.bufsize {
					buf.WriteTo(w.w)
					buf.Reset()
				}
			case <-time.After(1 * time.Second):
				if buf.Len() > 0 {
					buf.WriteTo(w.w)
					buf.Reset()
				}
			}
		}
	}()
}

type RotateWriter struct {
	LogDir string
	Name   string
	w      *os.File
	curLen int
	MaxLen int
}

func (w *RotateWriter) Write(b []byte) (int, error) {
	l := len(b)
	if w.curLen+l > w.MaxLen {
		w.rotate()
	}
	w.curLen += l
	return w.w.Write(b)
}

func (w *RotateWriter) rotate() {
	w.curLen = 0
	x, e := os.Create(w.LogDir + string(os.PathSeparator) + fmt.Sprintf("%s.%d", w.Name, time.Now().Unix()))

	if e != nil {
		panic(e)
	} else {
		if nil != w.w {
			w.w.Close()
		}
		w.w = x
	}
}

func NewLogger(logDir, logName string, level int) *Logger {
	rw := &RotateWriter{LogDir: logDir, Name: logName, MaxLen: 100 * 1024 * 1024}
	rw.rotate()
	c := make(chan []byte, 100)
	aw := &AsyncBuffWriter{c, 4096, rw}
	aw.run()
	l := log.New(aw, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	log := &Logger{l, level}
	return log
}
