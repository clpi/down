package fs

import (
	"io/fs"
	"log"
	"os"
	"path"
)

func LoadHome() error {
	if hp, he := os.UserHomeDir(); he == nil {
		h = hp
	} else {
		println("Error: ", he)
	}
	if tmpp, tmpe := os.UserCacheDir(); tmpe == nil {
		t = tmpp
	} else {
		return tmpe
	}
	if cfp, cfe := os.UserConfigDir(); cfe == nil {
		c = cfp
	} else {
		return cfe
	}
	home := path.Join(h, ".down/")
	data := path.Join(home, ".down/")
	log := path.Join(home, "log/")
	workspace := path.Join(home, "workspace/")

	down := path.Join(home, "config.down")
	configh := path.Join(home, "down.json")

	os.MkdirAll(data, 0o777)
	os.MkdirAll(log, 0o777)
	os.MkdirAll(workspace, 0o777)

	os.Create(down)
	os.Create(configh)

	if os.WriteFile(down, []byte{}, fs.ModePerm) != nil {
	}
	return nil
}

var (
	h string = "~/"
	t string = "~/tmp"
	c string = "~/.cache"
)

func ClearHome() error {
	if uh, err := os.UserHomeDir(); err == nil {
		return os.RemoveAll(path.Join(uh, ".down/"))
	} else {
		log.Println(err)
		return err
	}
}

func LoadOrCreateDir(pre string, ww ...string) (string, error) {
	var w = path.Join(pre, path.Join(ww...))
	var wi, e = os.Stat(w)
	if e != nil {
		log.Println(e)
		return "", e
	} else {
		if !wi.IsDir() {
			log.Println("Not a directory")
			if e := os.MkdirAll(w, 0o777); e != nil {
				return w, e
			}
		}
	}
	return w, nil
}
func LoadOrCreateWorkspace(w string, d ...string) (string, error) {
	ww, e := LoadOrCreateHome("workspace/", w)
	_, _ = LoadOrCreateHome(path.Join(ww, ".down/"))
	if e != nil {
		log.Println(e)
		return "", e
	}
	return LoadOrCreateDir(ww, d...)
}
func LoadOrCreateHome(d ...string) (string, error) {
	var uh, e = os.UserHomeDir()
	if e != nil {
		log.Println(e)
		return "", e
	}
	return LoadOrCreateDir(uh, d...)
}
