package files

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/andreasisnes/go-configuration-manager/modules"
	"github.com/andreasisnes/goflat"
	"github.com/fsnotify/fsnotify"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v3"
)

const (
	DefaultFile = "settings.json"
)

type Options struct {
	modules.Options
	File string
}

type Module struct {
	modules.ModuleBase
	FileOptions   Options
	Configuration map[string]interface{}
	Content       []byte
	WaitGroup     sync.WaitGroup
	QuitC         chan interface{}
}

func New(options *Options) modules.Module {
	if options == nil {
		options = &Options{
			File: DefaultFile,
		}
	}

	source := &Module{
		ModuleBase:    *modules.NewSourceBase(&options.Options),
		FileOptions:   *options,
		Configuration: make(map[string]interface{}),
		QuitC:         make(chan interface{}),
		WaitGroup:     sync.WaitGroup{},
	}

	if source.FileOptions.SentinelOptions != nil || source.Options.ReloadOnChange {
		go source.watcher()
	}

	return source
}

func (source *Module) GetRefreshedValue(key string) interface{} {
	return nil
}

func (source *Module) Deconstruct() {
	source.QuitC <- struct{}{}
}

func (source *Module) Load() {
	if fileExists(source.FileOptions.File) {
		if content, err := os.ReadFile(source.FileOptions.File); err != nil {
			panic(err)
		} else {
			source.Content = content
			extension := strings.ToLower(path.Ext(source.FileOptions.File))
			switch extension {
			case ".json":
				source.unmarshal(json.Unmarshal)
			case ".yml", ".yaml":
				source.unmarshal(yaml.Unmarshal)
			case ".toml":
				source.unmarshal(toml.Unmarshal)
			default:
				if !source.FileOptions.Optional {
					log.Fatalf("'%s' is not a <.json, yml, yaml, toml> file", path.Base(source.FileOptions.File))
				}
			}
		}
	} else {
		if !source.FileOptions.Optional {
			panic(os.ErrNotExist)
		}
	}
}

func (source *Module) watcher() {
	source.WaitGroup.Add(1)
	defer source.WaitGroup.Done()

	watcher, shouldReturn := source.createWatcher()
	if shouldReturn {
		return
	}
	defer watcher.Close()

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				source.NotifyDirtyness(source)
			}
		case err := <-watcher.Errors:
			if source.FileOptions.Optional {
				panic(err)
			}
			return
		case <-source.QuitC:
			return
		}
	}
}

func (source *Module) createWatcher() (*fsnotify.Watcher, bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		if !source.FileOptions.Optional {
			panic(err)
		}

		return nil, true
	}

	if !fileExists(source.FileOptions.File) {
		if !source.FileOptions.Optional {
			panic(os.ErrNotExist)
		}

		return nil, true
	}

	if err = watcher.Add(source.FileOptions.File); err != nil {
		if !source.FileOptions.Optional {
			panic(os.ErrNotExist)
		}

		return nil, true
	}

	return watcher, false
}

func (source *Module) unmarshal(fn func([]byte, interface{}) error) error {
	if err := fn(source.Content, &source.Configuration); err != nil {
		return err
	}

	source.Flatmap = goflat.Map(source.Configuration, &goflat.Options{
		Delimiter: goflat.DefaultDelimiter,
		Fold:      goflat.UpperCaseFold,
	})

	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
