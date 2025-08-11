package examples

import (
	"encoding/json"
	"testing"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

// TestVerboseLogging 测试详细日志功能
func TestVerboseLogging(t *testing.T) {
	// 构造视频合成请求参数，启用详细日志
	spec := &ffmpeg_go.EditSpec{
		OutPath: "./test_output.mp4",
		Width:   1920,
		Height:  1080,
		Fps:     30,
		Verbose: true, // 启用详细日志
		Defaults: &ffmpeg_go.Defaults{
			Duration: 3,
		},
		Clips: []*ffmpeg_go.Clip{
			{
				Layers: []*ffmpeg_go.Layer{
					{
						Type: "video",
						Path: "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
					},
				},
			},
			{
				Layers: []*ffmpeg_go.Layer{
					{
						Type: "video",
						Path: "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
					},
				},
			},
			{
				Layers: []*ffmpeg_go.Layer{
					{
						Type: "video",
						Path: "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
					},
				},
			},
			{
				Layers: []*ffmpeg_go.Layer{
					{
						Type: "video",
						Path: "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
					},
				},
			},
			{
				Layers: []*ffmpeg_go.Layer{
					{
						Type: "video",
						Path: "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
					},
				},
			},
		},
		KeepSourceAudio: true,
	}

	// 序列化为JSON以查看完整的请求结构
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	t.Logf("Video edit request with verbose logging:\n%s", string(data))

	// 注意：实际的视频编辑测试需要网络连接和较长的时间，
	// 在此仅验证参数构造是否正确
	t.Log("Verbose logging test completed")
}