package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/service"
)

type CaptureHandler struct {
	captureService *service.CaptureService
}

func NewCaptureHandler(captureService *service.CaptureService) *CaptureHandler {
	return &CaptureHandler{captureService: captureService}
}

// Upload handles a single fingerprint capture upload.
// Expects multipart/form-data with image file + metadata fields.
func (h *CaptureHandler) Upload(ctx *gin.Context) {
	// Parse multipart form — 10MB max
	if err := ctx.Request.ParseMultipartForm(10 << 20); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	// Read image file from multipart
	file, _, err := ctx.Request.FormFile("image")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Image file is required"})
		return
	}
	defer file.Close()

	// Read image bytes
	imageBytes := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			imageBytes = append(imageBytes, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Parse metadata fields from form
	nfiq2Score, _ := strconv.ParseFloat(ctx.Request.FormValue("nfiq2_score"), 64)
	blurScore, _ := strconv.ParseFloat(ctx.Request.FormValue("blur_score"), 64)
	brightnessScore, _ := strconv.ParseFloat(ctx.Request.FormValue("brightness_score"), 64)
	glareScore, _ := strconv.ParseFloat(ctx.Request.FormValue("glare_score"), 64)
	attemptCount, _ := strconv.Atoi(ctx.Request.FormValue("attempt_count"))
	degradedFlag, _ := strconv.ParseBool(ctx.Request.FormValue("degraded_flag"))

	req := model.CaptureRequest{
		SessionID:           ctx.Request.FormValue("session_id"),
		ResidentPseudonymID: ctx.Request.FormValue("resident_pseudonym_id"),
		OperatorID:          ctx.Request.FormValue("operator_id"),
		FingerType:          ctx.Request.FormValue("finger_type"),
		Hand:                ctx.Request.FormValue("hand"),
		Nfiq2Score:          nfiq2Score,
		BlurScore:           blurScore,
		BrightnessScore:     brightnessScore,
		GlareScore:          glareScore,
		AttemptCount:        attemptCount,
		DegradedFlag:        degradedFlag,
		ImageChecksum:       ctx.Request.FormValue("image_checksum"),
		CameraModel:         ctx.Request.FormValue("camera_model"),
		CameraResolution:    ctx.Request.FormValue("camera_resolution"),
		DeviceModel:         ctx.Request.FormValue("device_model"),
	}

	// Validate required fields
	if req.SessionID == "" || req.ResidentPseudonymID == "" ||
		req.OperatorID == "" || req.FingerType == "" || req.Hand == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	response, err := h.captureService.Upload(req, imageBytes)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// BatchUpload handles multiple pending captures.
// Each capture is a separate multipart request — batch sends them sequentially.
func (h *CaptureHandler) BatchUpload(ctx *gin.Context) {
	if err := ctx.Request.ParseMultipartForm(50 << 20); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	form := ctx.Request.MultipartForm
	files := form.File["images"]
	count := len(files)

	if count == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No images provided"})
		return
	}

	var responses []model.CaptureResponse

	for i := 0; i < count; i++ {
		idx := strconv.Itoa(i)

		file, err := files[i].Open()
		if err != nil {
			continue
		}
		defer file.Close()

		imageBytes := make([]byte, 0)
		buf := make([]byte, 1024)
		for {
			n, err := file.Read(buf)
			if n > 0 {
				imageBytes = append(imageBytes, buf[:n]...)
			}
			if err != nil {
				break
			}
		}

		nfiq2Score, _ := strconv.ParseFloat(getFormValue(form.Value, "nfiq2_score_"+idx), 64)
		blurScore, _ := strconv.ParseFloat(getFormValue(form.Value, "blur_score_"+idx), 64)
		brightnessScore, _ := strconv.ParseFloat(getFormValue(form.Value, "brightness_score_"+idx), 64)
		glareScore, _ := strconv.ParseFloat(getFormValue(form.Value, "glare_score_"+idx), 64)
		attemptCount, _ := strconv.Atoi(getFormValue(form.Value, "attempt_count_"+idx))
		degradedFlag, _ := strconv.ParseBool(getFormValue(form.Value, "degraded_flag_"+idx))

		req := model.CaptureRequest{
			SessionID:           getFormValue(form.Value, "session_id_"+idx),
			ResidentPseudonymID: getFormValue(form.Value, "resident_pseudonym_id_"+idx),
			OperatorID:          getFormValue(form.Value, "operator_id_"+idx),
			FingerType:          getFormValue(form.Value, "finger_type_"+idx),
			Hand:                getFormValue(form.Value, "hand_"+idx),
			Nfiq2Score:          nfiq2Score,
			BlurScore:           blurScore,
			BrightnessScore:     brightnessScore,
			GlareScore:          glareScore,
			AttemptCount:        attemptCount,
			DegradedFlag:        degradedFlag,
			ImageChecksum:       getFormValue(form.Value, "image_checksum_"+idx),
			CameraModel:         getFormValue(form.Value, "camera_model_"+idx),
			CameraResolution:    getFormValue(form.Value, "camera_resolution_"+idx),
			DeviceModel:         getFormValue(form.Value, "device_model_"+idx),
		}

		response, err := h.captureService.Upload(req, imageBytes)
		if err != nil {
			continue
		}
		responses = append(responses, *response)
	}

	ctx.JSON(http.StatusCreated, responses)
}

// getFormValue safely reads a form value from multipart form map
func getFormValue(values map[string][]string, key string) string {
	if vals, ok := values[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}
