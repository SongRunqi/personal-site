package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxUploadSize = 10 << 20 // 10 MB

// 允许的图片类型;SVG 可携带脚本,不收。
var imageExt = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// handleAdminUpload 接收 multipart 图片,存到 DATA_DIR/uploads/YYYY/MM/,
// 返回可直接写进 Markdown 的 URL。
func (s *server) handleAdminUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "没有收到文件(最大 10 MB)"})
		return
	}
	defer file.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	contentType := http.DetectContentType(head[:n])
	ext, ok := imageExt[contentType]
	if !ok {
		writeJSON(w, 422, map[string]string{"error": "只支持 PNG / JPEG / GIF / WebP 图片"})
		return
	}

	base := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	base = strings.Trim(unsafeName.ReplaceAllString(base, "-"), "-")
	if base == "" {
		base = "img"
	}
	if len(base) > 40 {
		base = base[:40]
	}
	rb := make([]byte, 4)
	rand.Read(rb)
	name := fmt.Sprintf("%s-%s%s", base, hex.EncodeToString(rb), ext)

	rel := filepath.Join(time.Now().Format("2006"), time.Now().Format("01"))
	dir := filepath.Join(s.dataDir, "uploads", rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("建上传目录失败:%v", err)
		writeJSON(w, 500, map[string]string{"error": "保存失败"})
		return
	}
	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		log.Printf("建文件失败:%v", err)
		writeJSON(w, 500, map[string]string{"error": "保存失败"})
		return
	}
	defer dst.Close()
	if _, err := dst.Write(head[:n]); err == nil {
		_, err = io.Copy(dst, file)
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "写入失败"})
		return
	}
	writeJSON(w, 201, map[string]string{"url": "/uploads/" + filepath.ToSlash(filepath.Join(rel, name))})
}

// uploadsHandler 提供 /uploads/ 下的静态图片。
func (s *server) uploadsHandler() http.Handler {
	fs := http.FileServer(http.Dir(filepath.Join(s.dataDir, "uploads")))
	return http.StripPrefix("/uploads/", fs)
}
