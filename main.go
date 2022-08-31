package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	outputDir   string
	productName string
	eLanguage   bool
)

func getCurrentPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	return filepath.Dir(path)
}

func makeDirEmpty(path string) error {
	f, err := os.Stat(path)
	if err != nil { //文件不存在
		return os.MkdirAll(path, 0750)
	}

	if f.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, file := range entries {
			fileName := filepath.Join(path, file.Name())
			os.RemoveAll(fileName)
		}
		return nil
	}

	return fmt.Errorf("%s is not a dir", path)
}

func main() {

	flag.StringVar(&outputDir, "o", "./target/yaml", "the output dir")
	flag.BoolVar(&eLanguage, "e", false, "english version")
	flag.StringVar(&productName, "product", "", "the product name, e.g. ECS, VPC. If not specified, all products will be scaned.")
	flag.Parse()

	outputPath, _ := filepath.Abs(outputDir)
	currPath := getCurrentPath()
	if !strings.Contains(outputPath, currPath) || (outputPath == currPath) {
		fmt.Printf("[ERROR] the output dir must be a sub-dir of the current dir, not %s\n", outputPath)
		os.Exit(1)
	}
	fmt.Printf("the output dir is %s\n", outputPath)

	err := makeDirEmpty(outputPath)
	if err != nil {
		fmt.Printf("[ERROR] failed to empty the output dir %s: %s\n", outputPath, err)
		os.Exit(2)
	}

	region := os.Getenv("HW_REGION")
	if productName == "" {
		ScanAllAPIs(outputPath, region)
	} else {
		ScanProductAPIs(outputPath, productName, region)
	}
}
