package core

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type FileInfo struct {
	Name string   `json:"name"`
	Path string   `json:"path"`
	Tags []string `json:"tags"`
	Ext  string   `json:"ext"`
}

type MetaData struct {
	LastUpdate time.Time `json:"last_update"`
	CreateTime time.Time `json:"create_time"`
}

type SearchParams struct {
	Name string
	Tag  string
	Ext  string
}

type IndexCreator struct {
	Config        *ScanConfig
	ignoreNameMap map[string]bool
	extMap        map[string]bool
	tagMap        map[string]bool
}

func InitIndex(cfg *Config) error {
	ic := NewIndexCreator(&cfg.ScanConfig)
	return ic.Create(cfg.SrcRoot, cfg.DataRoot)
}

func NewIndexCreator(cfg *ScanConfig) *IndexCreator {
	ic := &IndexCreator{}
	ic.Config = cfg

	ic.ignoreNameMap = map[string]bool{}
	for _, v := range cfg.IgnoreName {
		ic.ignoreNameMap[v] = true
	}

	ic.extMap = map[string]bool{}
	for _, v := range cfg.IgnoreExt {
		ic.extMap[v] = true
	}

	ic.tagMap = map[string]bool{}
	return ic
}

// 扫描目录
func (ic *IndexCreator) Create(root, dataRoot string) error {
	meta := ic.getMeta(dataRoot)

	// 修改时间没有变化
	if fi, err := os.Stat(root); err == nil {
		if fi.ModTime() == meta.LastUpdate {
			return nil
		}

		meta.CreateTime = time.Now()
		meta.LastUpdate = fi.ModTime()
	}

	if err := ic.createIndexFile(root, dataRoot); err != nil {
		return err
	}

	if err := ic.createExtListFile(dataRoot); err != nil {
		return err
	}

	if err := ic.createTagListFile(dataRoot); err != nil {
		return err
	}

	if err := writeJsonFile(path.Join(dataRoot, ic.Config.MetaFile), meta); err != nil {
		return err
	}
	return nil
}

func (ic *IndexCreator) createIndexFile(root, dataRoot string) error {
	reg, err := regexp.Compile(ic.Config.TagExpr)
	if err != nil {
		log.Panicln("compile reg expr error", ic.Config.TagExpr, err)
		return err
	}

	index := path.Join(dataRoot, ic.Config.IndexFile)
	if err := os.MkdirAll(dataRoot, os.ModePerm); err != nil {
		log.Panicln("mkdir all", dataRoot, err)
		return err
	}

	idx, err := os.OpenFile(index, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)

	if err != nil {
		return err
	}

	absRootPath, _ := filepath.Abs(root)

	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		name := info.Name()
		ext := getExt(name)

		if ic.ignoreNameMap[name] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if ic.extMap[ext] {
			return nil
		}

		ic.extMap[ext] = false

		tags, err := parseTag(name, reg)
		if err != nil {
			return err
		}

		for _, v := range tags {
			ic.tagMap[v] = true
		}

		filePath := strings.TrimPrefix(path, absRootPath)
		filePath = normalizePath(filePath)
		v := FileInfo{
			Name: name,
			Path: filePath,
			Tags: tags,
			Ext:  ext,
		}

		if b, err := json.Marshal(v); err != nil {
			return err
		} else {
			if _, err := idx.Write(b); err != nil {
				return err
			}
			if _, err := idx.Write([]byte{'\n'}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (ic *IndexCreator) getMeta(dataRoot string) *MetaData {
	meta := &MetaData{}
	filename := path.Join(dataRoot, ic.Config.MetaFile)
	b, err := os.ReadFile(filename)
	if err != nil {
		return meta
	}

	if err := json.Unmarshal(b, meta); err != nil {
		return meta
	}
	return meta
}

func (ic *IndexCreator) createExtListFile(dataRoot string) error {
	filename := path.Join(dataRoot, ic.Config.ExtListFile)
	if err := writeJsonFile(filename, ic.extMap); err != nil {
		return err
	}
	return nil
}

func (ic *IndexCreator) createTagListFile(dataRoot string) error {
	filename := path.Join(dataRoot, ic.Config.TagListFile)
	tags := []string{}
	for k := range ic.tagMap {
		tags = append(tags, k)
	}
	if err := writeJsonFile(filename, tags); err != nil {
		return err
	}
	return nil
}

func writeJsonFile(filename string, v interface{}) error {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	if data, err := json.Marshal(v); err != nil {
		return err
	} else {
		if _, err := f.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func parseTag(name string, reg *regexp.Regexp) ([]string, error) {
	matches := reg.FindAllStringSubmatch(name, -1)
	tags := []string{}
	for _, m := range matches {
		tags = append(tags, m[1])
	}
	return tags, nil
}

func normalizePath(filename string) string {
	return strings.ReplaceAll(filename, "\\", "/")
}

func getExt(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) <= 1 {
		return ""
	}
	return strings.ToLower(ext[1:])
}
