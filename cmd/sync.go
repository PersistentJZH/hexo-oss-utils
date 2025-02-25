/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/spf13/cobra"
)

var (
	endpoint        string
	accessKeyID     string
	accessKeySecret string
	bucketName      string
	directory       string
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("sync called")
		generateLinks()
	},
}

func init() {

	syncCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "OSS endpoint (required)")
	syncCmd.Flags().StringVarP(&accessKeyID, "access-key-id", "i", "", "OSS access key ID (required)")
	syncCmd.Flags().StringVarP(&accessKeySecret, "access-key-secret", "s", "", "OSS access key secret (required)")
	syncCmd.Flags().StringVarP(&bucketName, "bucket", "b", "", "OSS bucket name (required)")
	syncCmd.Flags().StringVarP(&directory, "directory", "d", "", "Target directory in OSS bucket (required)")

	syncCmd.MarkFlagRequired("endpoint")
	syncCmd.MarkFlagRequired("access-key-id")
	syncCmd.MarkFlagRequired("access-key-secret")
	syncCmd.MarkFlagRequired("bucket")
	syncCmd.MarkFlagRequired("directory")
	rootCmd.AddCommand(syncCmd)

}

func generateLinks() {
	// Initialize OSS client
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		fmt.Printf("Error initializing OSS client: %v\n", err)
		return
	}

	// Get bucket reference
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		fmt.Printf("Error accessing bucket: %v\n", err)
		return
	}

	// Process directory path
	normalizedDir := normalizeDirectoryPath(directory)

	// Collect all objects
	var objects []string
	marker := ""
	for {
		lsRes, err := bucket.ListObjects(oss.Marker(marker), oss.Prefix(normalizedDir))
		if err != nil {
			fmt.Printf("Error listing objects: %v\n", err)
			return
		}

		for _, object := range lsRes.Objects {
			if !strings.HasSuffix(object.Key, "/") { // Exclude directories
				objects = append(objects, object.Key)
			}
		}

		if lsRes.IsTruncated {
			marker = lsRes.NextMarker
		} else {
			break
		}
	}
	data := make(map[string][]string)

	// Generate markdown content
	var buf bytes.Buffer
	buf.WriteString("# OSS Download Links\n\n")
	for _, key := range objects {
		module := strings.Split(key, "/")[1]
		link := formatOSSUrl(key)
		data[module] = append(data[module], link)
	}

	// 写item文件
	err = writeItemFile(data)
	if err != nil {
		fmt.Printf("Error write item file : %v\n", err)
		return
	}

	overviewFileData := make([]OverviewFileDataObj, 0, len(data))
	for k, items := range data {
		var overviewFileDataObj OverviewFileDataObj
		overviewFileDataObj.Path = "/" + k
		overviewFileDataObj.PaperPost = items[0]
		overviewFileDataObj.Name = k
		overviewFileData = append(overviewFileData, overviewFileDataObj)
	}

	// 写overview文件
	err = writeOverviewFile(overviewFileData)
	if err != nil {
		fmt.Printf("Error write item file : %v\n", err)
		return
	}

}

type OverviewFileDataObj struct {
	Path      string
	PaperPost string
	Name      string
}

func writeOverviewFile(data []OverviewFileDataObj) error {
	const tpl = `
---
title: gallery
date: {{.Time}}
---
<div class="gallery-group-main">
{{range .Data}}
{% galleryGroup '{{.Name}}' '收藏的一些壁纸' '{{.Path}}' {{.PaperPost}} %}
{{end}}
</div>
`
	filePrefix := "gallery"
	fileName := filePrefix + "/index.md"
	t, err := template.New("test").Parse(tpl)
	if err != nil {
		return err
	}

	// 创建目录
	err = os.MkdirAll(filePrefix, os.ModePerm)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	// 准备模板数据
	templateData := struct {
		Time string
		Data []OverviewFileDataObj
	}{
		Time: time.Now().Format("2006-01-02 15:04:05"),
		Data: data,
	}
	err = t.Execute(file, templateData)
	if err != nil {
		return err
	}
	file.Close()

	fmt.Printf("Successfully write all item files")
	return nil
}

func writeItemFile(data map[string][]string) error {
	const tpl = `
---
title: gallery/wallpaper
date: {{.Time}}
---
{% gallery %}
{{range .Data}}
{{.}}
{{end}}
{% endgallery %}
`
	for filePrefix, items := range data {
		cloneItems := make([]string, 0, len(items))
		for _, item := range items {
			cloneItems = append(cloneItems, fmt.Sprintf("- ![](%s)\n", item))
		}
		fileName := filePrefix + "/index.md"
		t, err := template.New("test").Parse(tpl)
		if err != nil {
			return err
		}

		// 创建目录
		err = os.MkdirAll(filePrefix, os.ModePerm)
		if err != nil {
			panic(err)
		}

		file, err := os.Create(fileName)
		if err != nil {
			return err
		}
		// 准备模板数据
		templateData := struct {
			Time string
			Data []string
		}{
			Time: time.Now().Format("2006-01-02 15:04:05"),
			Data: cloneItems,
		}
		err = t.Execute(file, templateData)
		if err != nil {
			return err
		}
		file.Close()
		fmt.Printf("Successfully write %s with %d links\n", fileName, len(items))
	}
	fmt.Printf("Successfully write all item files")
	return nil
}
func normalizeDirectoryPath(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}

func getFileName(key, dir string) string {
	return strings.TrimPrefix(key, dir)
}

func formatOSSUrl(key string) string {
	return fmt.Sprintf("https://%s.%s/%s", bucketName, endpoint, key)
}
