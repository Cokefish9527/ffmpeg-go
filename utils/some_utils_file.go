package utils

func HandlePanic() {
    if r := recover(); r != nil {
        // 处理panic
    }
}
