package admin

import (
	"net/http"
	"strings"

	"storecore/internal/pkg/response"
	"storecore/internal/storage"

	"github.com/gin-gonic/gin"
)

// maxImageSize 保留兼容旧引用
const maxImageSize = maxImageUploadSize

type UploadHandler struct {
	store storage.Storage
}

func NewUploadHandler(store storage.Storage) *UploadHandler {
	return &UploadHandler{store: store}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "file required")
		return
	}
	kind, maxSize, ok := classifyUploadFile(file)
	if !ok {
		response.Fail(c, http.StatusBadRequest, "unsupported file type (image/video)")
		return
	}
	if file.Size > maxSize {
		if kind == "video" {
			response.Fail(c, http.StatusBadRequest, "video too large (max 100MB)")
		} else {
			response.Fail(c, http.StatusBadRequest, "image too large (max 10MB)")
		}
		return
	}
	subdir := c.DefaultPostForm("subdir", "stores")
	subdir = strings.Trim(subdir, "/")
	if subdir == "" {
		subdir = "stores"
	}
	url, err := h.store.Upload(file, subdir)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"url": url, "mediaType": kind})
}
