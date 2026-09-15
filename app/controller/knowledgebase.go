package controller

import (
	"io"
	"net/http"
	"strconv"

	"github.com/David3310273/go-agent/app/services"
	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
)

const knowledgeServiceKey = "knowledgeService"

// HandleCreateKnowledge handles POST /internal/v1/knowledgebase
// [auto-added] accepts multipart/form-data with file upload, contentType and domain fields.
// storageType is validated in middleware.
func HandleCreateKnowledge(c *gin.Context) {
	ks := c.MustGet(knowledgeServiceKey).(*services.MilvusKnowledgeService)

	// get contentType field
	contentType := c.PostForm("contentType")
	if contentType == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "contentType is required",
		})
		return
	}

	// get domain field
	domain := c.PostForm("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "domain is required",
		})
		return
	}

	// get uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "file is required",
			Data:    err.Error(),
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "failed to open file",
			Data:    err.Error(),
		})
		return
	}
	defer file.Close()

	// read file data
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "failed to read file",
			Data:    err.Error(),
		})
		return
	}

	req := &services.MilvusKnowledgeCreateRequest{
		StorageType: services.StorageTypeMilvus,
		ContentType: contentType,
		Domain:      domain,
		Data:        data,
		Filename:    fileHeader.Filename,
	}

	resp, diag := ks.CreateMilvusKnowledge(req)
	if diag != nil {
		c.JSON(http.StatusInternalServerError, diag)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// HandleGetKnowledge handles GET /internal/v1/knowledgebase
// [auto-added] searches by keyword, filename, contentType with domain.
// storageType is validated in middleware.
func HandleGetKnowledge(c *gin.Context) {
	ks := c.MustGet(knowledgeServiceKey).(*services.MilvusKnowledgeService)

	// get query parameters
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "domain is required",
		})
		return
	}

	contentType := c.Query("contentType")
	filename := c.Query("filename")
	keyword := c.Query("keyword")

	// at least one search criterion is required
	if keyword == "" && filename == "" && contentType == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "at least one of keyword, filename, or contentType is required",
		})
		return
	}

	topK := 10
	if topKStr := c.Query("topK"); topKStr != "" {
		if n, err := strconv.Atoi(topKStr); err == nil && n > 0 {
			topK = n
		}
	}

	req := &services.MilvusKnowledgeGetRequest{
		StorageType: services.StorageTypeMilvus,
		Domain:      domain,
		ContentType: contentType,
		Filename:    filename,
		Keyword:     keyword,
		TopK:        topK,
	}

	resp, diag := ks.GetMilvusKnowledge(req)
	if diag != nil {
		status := http.StatusInternalServerError
		if diag.Level == core.SeverityWarn {
			status = http.StatusNotFound
		}
		c.JSON(status, diag)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleUpdateKnowledge handles PUT /internal/v1/knowledgebase
// Update is implemented as delete + create: delete existing entries by filename/contentType, then create new one.
// [auto-added] accepts multipart/form-data with file upload, contentType and domain fields.
// storageType is validated in middleware.
func HandleUpdateKnowledge(c *gin.Context) {
	ks := c.MustGet(knowledgeServiceKey).(*services.MilvusKnowledgeService)

	// get contentType field
	contentType := c.PostForm("contentType")
	if contentType == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "contentType is required",
		})
		return
	}

	// get domain field
	domain := c.PostForm("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "domain is required",
		})
		return
	}

	// get uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "file is required",
			Data:    err.Error(),
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "failed to open file",
			Data:    err.Error(),
		})
		return
	}
	defer file.Close()

	// read file data
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "failed to read file",
			Data:    err.Error(),
		})
		return
	}

	// [auto-added] delete existing entries by filename and contentType before creating new one
	deleteReq := &services.MilvusKnowledgeDeleteRequest{
		StorageType: services.StorageTypeMilvus,
		Domain:      domain,
		ContentType: contentType,
		Filename:    fileHeader.Filename,
	}
	if _, diag := ks.DeleteMilvusKnowledge(deleteReq); diag != nil {
		// log warning but continue to create new entry
		c.JSON(http.StatusInternalServerError, diag)
		return
	}

	// create new entry
	createReq := &services.MilvusKnowledgeCreateRequest{
		StorageType: services.StorageTypeMilvus,
		ContentType: contentType,
		Domain:      domain,
		Data:        data,
		Filename:    fileHeader.Filename,
	}

	resp, diag := ks.CreateMilvusKnowledge(createReq)
	if diag != nil {
		c.JSON(http.StatusInternalServerError, diag)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleDeleteKnowledge handles DELETE /internal/v1/knowledgebase
// [auto-added] deletes by filename and contentType with domain filter.
// storageType is validated in middleware.
func HandleDeleteKnowledge(c *gin.Context) {
	ks := c.MustGet(knowledgeServiceKey).(*services.MilvusKnowledgeService)

	// get query parameters
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "domain is required",
		})
		return
	}

	contentType := c.Query("contentType")
	filename := c.Query("filename")

	// at least one of filename or contentType is required
	if filename == "" && contentType == "" {
		c.JSON(http.StatusBadRequest, core.Diagnostic{
			Code:    core.MessageCodeSystemError,
			Level:   core.SeverityError,
			Message: "at least one of filename or contentType is required",
		})
		return
	}

	req := &services.MilvusKnowledgeDeleteRequest{
		StorageType: services.StorageTypeMilvus,
		Domain:      domain,
		ContentType: contentType,
		Filename:    filename,
	}

	resp, diag := ks.DeleteMilvusKnowledge(req)
	if diag != nil {
		c.JSON(http.StatusInternalServerError, diag)
		return
	}

	c.JSON(http.StatusOK, resp)
}
