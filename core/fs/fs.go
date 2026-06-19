package fs

import (
	// "io/fs"
	// log "log"
	"os"
	"path"
	"strings"
)

var (
	uhome, uhomerr      = os.UserHomeDir()
	uconfig, uconfigerr = os.UserConfigDir()
	ucache, ucacheerr   = os.UserCacheDir()
	utemp               = os.TempDir()
)
var (
	lsp       = path.Join(uhome, ".down/", "lsp/")
	data      = path.Join(uhome, ".down/", "data/")
	dump      = path.Join(uhome, ".down/", "log/")
	home      = path.Join(uhome, ".down/")
	conf      = path.Join(uconfig, "down/")
	cache     = path.Join(ucache, "down/")
	llog      = path.Join(utemp, "down/")
	tmp       = path.Join(utemp, "down/")
	ws        = path.Join(home, "workspace/")
	workspace = ws
	config    = path.Join(conf, "downrc")
	doc       = path.Join(conf, "down.dd")
)

var (
	wsDown = path.Join(ws)
)
var (
	confRoot = path.Join(conf, "down.toml")
)

// type Map map[string]map[string]any
// type MapInterface map[string]interface{}
// type MapStruct map[string]struct{}
// type EnvVals map[string]string
// type MapList map[string] []string
// type MapMap map[string][]map[string]string
// type EnvMap map[string]interface{
//   map[string]interface{

//   }
// }
// type EnvMap map[string]any
// type EnvMap map[string]map[string]any{
//   `DOOM_WORKSPACE` map[string]interface{}

// }
// //     `DOOM_WORKSPACE` map[string]interface{
//   }
// }
// type EnvLists map[string][]string{
//   `DOOM_WORKSPACE`: []string
// } )
// type EnvItems map[string]struct {
//   DOWN_WORKSPACE []string
// }
// type EnvStruct map[string]map[string]struct {
//   DOWN_WORKSPACE []string

// }
// type Env MapList {
//   DOWN_WORKSPACE []string
// }

func Workspaces() []string {
	// w := struct { string: []string } { }
	// fs.ReadDir(workspace)
	// } { }
	// fs.ReadDir(workspace)
	// os.Chdir(workspace)
	return []string{}
}
func Workspace(w string) string {
	println("HOME", uhome, home)
	wr := path.Join(ws, w)
	os.MkdirAll(wr, 0777)
	os.Setenv("DOOM_WORKSPACES_DIR", wr)
	os.Setenv("DOOM_WORKSPACES", strings.Join([]string{os.Getenv("DOOM_WORKSPACES"), w}, ":"))
	// other := os.Getenv("DOWN_WORKSPACES")
	// println("other, w", other, w)
	println("WOKRSP", strings.Join([]string{os.Getenv("DOWN_WORKSPACES"), w}, ":"))
	// os.Setenv("DOWN_WORKSPACES", strings.Join([]string{other, w}, ":"))
	// println("WOKRSP", strings.Join([]string{other, w}, ":"))
	os.Setenv("DOOM_CACHE_DIR", cache)
	os.Setenv("DOOM_CONFIG_DIR", config)
	os.Setenv("DOOM_LOG_DIR", llog)
	os.Setenv("DOOM_DATA_DIR", path.Join(home, "data"))
	os.Setenv("DOOM_RUNTIME_DIR", path.Join(home, "rt"))
	os.MkdirAll(config, 0777)
	os.MkdirAll(cache, 0777)
	os.MkdirAll(tmp, 0777)
	os.MkdirAll(llog, 07777)
	wd := path.Join(wr, ".down/")
	os.MkdirAll(wd, 07777)

	os.Chdir(wr)
	// p, _ := os.StartProcess("git", []string{"init"}, nil)
	// p.Wait()

	wc := path.Join(wr, strings.Join([]string{w, ".dd"}, ""))
	println("wc", wc)
	_, _ = os.Create(wc)

	return wr
}
func WorkspaceRmAll() {
	os.RemoveAll(workspace)
}
func WorkspaceRm(w string) {
	wp := path.Join(workspace, w)
	_, err := os.ReadDir(wp)
	if err != nil {
		os.RemoveAll(wp)
	}
}
func WorkspaceChildren(w string) ([]os.DirEntry, error) {
	wp := path.Join(workspace, w)
	wd, err := os.ReadDir(wp)
	if err != nil {
		return wd, nil
	}
	return nil, err

}
func MkDirFile(pre string, dir string, file string) string {
	d := path.Join(pre, dir)
	f := path.Join(d, file)
	os.Mkdir(d, 0777)
	os.Create(f)
	return f
}
func MkDirDir(pre string, d1 string, d2 string) string {
	di1 := path.Join(pre, d1) // "~/.down"
	di2 := path.Join(di1, d2) // "~/.down/workspace"
	os.MkdirAll(di2, 0777)
	return di1
}
func MkDirDirFile(pre string, d1 string, d2 string, file string) *os.File {
	d := MkDirDir(pre, d1, d2)
	f, _ := os.Create(path.Join(d, file))
	return f
}

// / ~/.down/workspace/
func MkDownDirsFile(pre string, dir string, sub string, f string) *os.File {
	return MkDirDirFile(home, dir, sub, f)
}
func MkDownFile(pre string, file string) string {
	return MkDirFile(pre, "down/", file)
}
func MkWorkspace(w string) {
	os.MkdirAll(path.Join(workspace, w), 0777)
}
func InitData() {
	os.MkdirAll(workspace, 0777)
}
