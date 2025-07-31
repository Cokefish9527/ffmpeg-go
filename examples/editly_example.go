package examples

import (
	"fmt"
	"log"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func ExampleEditly() {
	// 从JSON文件加载编辑规范
	spec := &ffmpeg.EditSpec{
		OutPath: "./examples/sample_data/editly_output.mp4",
		Width:   640,
		Height:  480,
		Fps:     25,
		Defaults: &ffmpeg.Defaults{
			Duration: 3,
			Transition: &ffmpeg.Transition{
				Duration: 0.5,
				Name:     "fade",
			},
		},
		Clips: []*ffmpeg.Clip{
			{
				Layers: []*ffmpeg.Layer{
					{
						Type: "video",
						Path: "./examples/sample_data/in1.mp4",
					},
				},
			},
			{
				Layers: []*ffmpeg.Layer{
					{
						Type: "image",
						Path: "./examples/sample_data/overlay.png",
					},
				},
			},
			{
				Layers: []*ffmpeg.Layer{
					{
						Type:     "title",
						Text:     "Hello, ffmpeg-go!",
						Color:    "blue",
						FontSize: 32,
					},
				},
			},
			{
				Layers: []*ffmpeg.Layer{
					{
						Type:  "fill-color",
						Color: "red",
					},
					{
						Type:  "title",
						Text:  "Red Background",
						Color: "white",
					},
				},
			},
		},
		KeepSourceAudio: true,
		Verbose:         true,
	}

	// 使用Editly编辑视频
	editly := ffmpeg.NewEditly(spec)
	err := editly.Edit()
	if err != nil {
		log.Fatalf("编辑视频时出错: %v", err)
	}

	fmt.Println("视频编辑完成!")
}

func ExampleEditlyFromFile() {
	// 从文件加载编辑规范
	editly, err := ffmpeg.FromFile("./examples/sample_data/editly_example.json")
	if err != nil {
		log.Fatalf("加载编辑规范时出错: %v", err)
	}

	// 编辑视频
	err = editly.Edit()
	if err != nil {
		log.Fatalf("编辑视频时出错: %v", err)
	}

	fmt.Println("从文件编辑视频完成!")
}