package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/StevenZack/gte/util"
	"github.com/StevenZack/tools/strToolkit"
	"golang.org/x/text/language"
)

type Config struct {
	Host         string            `json:"host"`
	Port         int               `json:"port"`
	Routes       []Route           `json:"routes"`
	NotFoundPage string            `json:"notFoundPage"`
	BlackList    []string          `json:"blackList"`
	ApiServer    string            `json:"apiServer"` //API server, e.g. "http://localhost:12300"
	Envs         map[string]Config `json:"envs"`      //customized environments
	Lang         struct {
		Dir        string `json:"dir"`        //language resources location
		Default    string `json:"default"`    //default language, e.g. 'zh-CN'
		KeyAsValue bool   `json:"keyAsValue"` //return key as value when request of default language comes
	} `json:"lang"` //language setup

	Root              string                       `json:"-"` //root directory of your project
	Env               string                       `json:"-"`
	InternalBlackList []string                     `json:"-"`
	Strs              map[string]map[string]string `json:"-"`
}
type Route struct {
	Path string `json:"path"`
	To   string `json:"to"`
}

const (
	CONFIG_FILE_NAME = "gte.config.json"
)

func LoadConfig(env, root string, port int) (Config, error) {
	v := Config{
		Env:  env,
		Host: "0.0.0.0",
		Port: port,
		Root: root,
		InternalBlackList: []string{
			"/" + CONFIG_FILE_NAME,
		},
		ApiServer: "http://localhost",
	}

	//gte.config.json
	b, e := ioutil.ReadFile(filepath.Join(root, CONFIG_FILE_NAME))
	if e != nil {
		if os.IsNotExist(e) {
			return v, nil
		}
		log.Println(e)
		return v, e
	}

	e = json.Unmarshal(b, &v)
	if e != nil {
		log.Println(e)
		return v, e
	}

	//handle envs
	if v.Envs != nil && env != "" {
		v1, ok := v.Envs[env]
		if !ok {
			return v, errors.New("No environment named '" + env + "'")
		}
		e := util.ReplaceFieldIND(&v, v1)
		if e != nil {
			return v, e
		}
	}

	//lang file check
	if v.Lang.Dir != "" {
		if v.Lang.Default == "" {
			return v, errors.New("'lang.dir' configure is set, but default language is not set. e.g. 'zh-HK'")
		}
		langDir := filepath.Join(v.Root, v.Lang.Dir)
		if _, e := os.Stat(langDir); os.IsNotExist(e) {
			return v, errors.New("The language directory '" + langDir + "' doesn't exist")
		}
		v.Strs = make(map[string]map[string]string)
		fs, e := ioutil.ReadDir(langDir)
		if e != nil {
			log.Println(e)
			return v, e
		}
		for _, f := range fs {
			if !strings.HasSuffix(f.Name(), util.LANG_FILE_EXT) {
				continue
			}
			lang := strToolkit.TrimEnd(f.Name(), util.LANG_FILE_EXT)
			_, e := language.Parse(lang)
			if e != nil {
				return v, errors.New("Invalid language resource name '" + f.Name() + "', e.g. 'zh-HK'" + util.LANG_FILE_EXT + " .https://www.unicode.org/reports/tr35/#Unicode_Language_and_Locale_Identifiers")
			}

			//load
			filepath := filepath.Join(langDir, f.Name())
			m, e := util.LoadJsonLangFile(filepath)
			if e != nil {
				log.Println(e)
				return v, fmt.Errorf("Reading language resource file '"+f.Name()+"' failed: %w", e)
			}
			v.Strs[lang] = m
		}

		if _, ok := v.Strs[v.Lang.Default]; !v.Lang.KeyAsValue && !ok {
			return v, errors.New("The default language resource file '" + v.Lang.Default + ".json' not found")
		}
	}
	return v, nil
}

func (r *Route) Params(uri string) map[string]string {
	ss1 := strings.Split(r.Path, "/")
	ss2 := strings.Split(uri, "/")
	m := make(map[string]string)
	for i := 0; i < len(ss1) && i < len(ss2); i++ {
		k := ss1[i]
		if k == "" {
			continue
		}
		if !strings.HasPrefix(k, ":") {
			continue
		}
		k = strToolkit.TrimStart(k, ":")

		v := ss2[i]
		m[k] = v
	}
	return m
}

func (r *Route) ParamPrefix() (string, bool) {
	i := strings.Index(r.Path, "/:")
	if i == -1 {
		return "", false
	}
	return r.Path[:i], true
}
