package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

var (
	outputDir string
	eLanguage bool
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

	ScanAllAPIs(outputPath)

}

func ScanAllAPIs(path string) error {
	var total, success int

	region := os.Getenv("HW_REGION")
	if region != "" {
		region = "cn-north-4"
	}

	groups, err := getAllProducts()
	if err != nil {
		fmt.Printf("[ERROR] querying all products: %s\n", err)
		return err
	}

	for i, g := range groups {
		fmt.Printf("[DEBUG] %d group name: %s\n", i, g.Name)
		catalog := strings.ReplaceAll(g.Name, " ", "_")
		os.Mkdir(filepath.Join(path, catalog), 0750)

		for j, p := range g.Products {
			fmt.Printf("\t[DEBUG] %d product name: %s(%s) / %d\n", j, p.Name, p.ProductShort, p.APICount)
			if p.APICount == 0 {
				continue
			}
			total += p.APICount

			apiInfos, err := getProductAPIs(p.ProductShort)
			if err != nil || len(apiInfos) == 0 {
				fmt.Printf("\t[WARN] failed to fetch APIs of %s: %s\n", p.ProductShort, err)
				continue
			}

			productDir := filepath.Join(path, catalog, p.ProductShort)
			os.Mkdir(productDir, 0750)

			for _, item := range apiInfos {
				detail, err := getAPIDetail(p.ProductShort, item.Name, region)
				if err != nil {
					fmt.Printf("\t[WARN] failed to fetch API detail: %s\n", err)
					continue
				}

				yamlFile := fmt.Sprintf("%s/%s.yaml", productDir, item.Name)
				if err := convertJSON2YAML(detail, yamlFile); err != nil {
					fmt.Printf("\t[WARN] failed to save yaml: %s\n", err)
					continue
				}
				success++
			}
		}
	}

	fmt.Printf("[DEBUG] total APIs: %d / %d\n", success, total)
	return nil
}

func convertJSON2YAML(body []byte, path string) error {
	var response interface{}
	err := json.Unmarshal(body, &response)
	if err != nil || response == nil {
		return fmt.Errorf("Unmarshal failed: %s", err)
	}

	yamlBytes, err := yaml.Marshal(response)
	if err != nil {
		return fmt.Errorf("Error marshaling into YAML from JSON: %s", err)
	}

	return writeYamlFile(yamlBytes, path)
}

func writeYamlFile(body []byte, file string) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(body); err != nil {
		return err
	}

	return nil
}
