package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
	"contactless-fingerprint-backend/internal/service"
)

type CaptureHandler struct {
	captureService *service.CaptureService
}

func NewCaptureHandler(captureService *service.CaptureService) *CaptureHandler {
	return &CaptureHandler{captureService: captureService}
}

var validFingerTypes = map[string]bool{
	"LEFT_THUMB":   true,
	"LEFT_INDEX":   true,
	"LEFT_MIDDLE":  true,
	"LEFT_RING":    true,
	"LEFT_LITTLE":  true,
	"RIGHT_THUMB":  true,
	"RIGHT_INDEX":  true,
	"RIGHT_MIDDLE": true,
	"RIGHT_RING":   true,
	"RIGHT_LITTLE": true,
}

var allowedFingerTypes = []string{
	"LEFT_THUMB", "LEFT_INDEX", "LEFT_MIDDLE", "LEFT_RING", "LEFT_LITTLE",
	"RIGHT_THUMB", "RIGHT_INDEX", "RIGHT_MIDDLE", "RIGHT_RING", "RIGHT_LITTLE",
}

var validHands = map[string]bool{
	"LEFT":  true,
	"RIGHT": true,
}

var allowedHands = []string{"LEFT", "RIGHT"}

// Upload handles a single fingerprint capture upload.
// Expects multipart/form-data with image file + metadata fields.
//
//	400 — missing required fields or invalid enum values (finger_type, hand)
//	404 — session_id does not exist
//	409 — this finger was already captured for this session
//	422 — scores out of valid range, or attempt_count <= 0
//	500 — unexpected error
func (h *CaptureHandler) Upload(ctx *gin.Context) {
	if err := ctx.Request.ParseMultipartForm(10 << 20); err != nil {
		respondError(ctx, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	file, _, err := ctx.Request.FormFile("image")
	if err != nil {
		respondError(ctx, http.StatusBadRequest, "Image file is required")
		return
	}
	defer file.Close()

	imageBytes := readAll(file)

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

	if statusCode, msg, data := validateCaptureRequest(req); msg != "" {
		if data != nil {
			respondErrorWithData(ctx, statusCode, msg, data)
		} else {
			respondError(ctx, statusCode, msg)
		}
		return
	}

	response, err := h.captureService.Upload(req, imageBytes)
	if err != nil {
		mapCaptureError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// batchItemValidationError describes a validation failure for one item in a batch.
type batchItemValidationError struct {
	Index   int    `json:"index"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// BatchUpload handles multiple pending captures in one multipart request.
// All items are validated before any are processed; if any fail validation the
// whole request is rejected with 400 and details of which indices failed.
//
// Metadata fields are indexed: session_id_0, finger_type_0, … session_id_1, …
// Image files are in the "images" file array, aligned by index.
//
//	400 — missing/invalid fields in one or more items (no items processed)
//	404 — session_id for an item does not exist
//	409 — duplicate capture for an item
//	500 — unexpected error
func (h *CaptureHandler) BatchUpload(ctx *gin.Context) {
	if err := ctx.Request.ParseMultipartForm(50 << 20); err != nil {
		log.Printf("ParseMultipartForm error: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	form := ctx.Request.MultipartForm
	log.Printf("Multipart form keys: %+v", form.Value)
	log.Printf("Multipart file keys: %+v", form.File)

	files := form.File["image"]
	log.Printf("Number of 'images' files found: %d", len(files))

	count := len(files)
	if count == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No images provided"})
		return
	}
	// ── Phase 1: validate all items before processing any ──────────────────
	type parsedItem struct {
		req        model.CaptureRequest
		imageBytes []byte
	}
	items := make([]parsedItem, 0, count)
	var valErrs []batchItemValidationError

	for i := 0; i < count; i++ {
		idx := strconv.Itoa(i)

		f, err := files[i].Open()
		if err != nil {
			valErrs = append(valErrs, batchItemValidationError{i, "image", "Could not read image file"})
			items = append(items, parsedItem{}) // placeholder
			continue
		}
		imgBytes := readAll(f)
		f.Close()

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

		if statusCode, msg, _ := validateCaptureRequest(req); msg != "" {
			field := captureValidationField(statusCode, req)
			valErrs = append(valErrs, batchItemValidationError{i, field, msg})
		}

		items = append(items, parsedItem{req: req, imageBytes: imgBytes})
	}

	if len(valErrs) > 0 {
		respondErrorWithData(ctx, http.StatusBadRequest, "Batch validation failed", valErrs)
		return
	}

	// ── Phase 2: process all validated items ───────────────────────────────
	var responses []model.CaptureResponse

	for i, item := range items {
		response, err := h.captureService.Upload(item.req, item.imageBytes)
		if err != nil {
			if errors.Is(err, repository.ErrDuplicateCapture) {
				// Data already exists on the server — skip without failing the batch.
				log.Printf("BatchUpload item %d skipped (already captured): finger_type=%s", i, item.req.FingerType)
				continue
			}
			log.Printf("BatchUpload item %d error: %v", i, err)
			mapCaptureError(ctx, err)
			return
		}
		responses = append(responses, *response)
	}

	ctx.JSON(http.StatusCreated, responses)
}

// validateCaptureRequest checks required fields, enum values, and score ranges.
// Returns (statusCode, errorMessage, extraData). If message is empty, the request is valid.
func validateCaptureRequest(req model.CaptureRequest) (int, string, interface{}) {
	if req.SessionID == "" {
		return http.StatusBadRequest, "session_id is required", nil
	}
	if req.ResidentPseudonymID == "" {
		return http.StatusBadRequest, "resident_pseudonym_id is required", nil
	}
	if req.OperatorID == "" {
		return http.StatusBadRequest, "operator_id is required", nil
	}

	if req.FingerType == "" {
		return http.StatusBadRequest, "finger_type is required",
			gin.H{"allowed_values": allowedFingerTypes}
	}
	if !validFingerTypes[req.FingerType] {
		return http.StatusBadRequest, "Invalid finger_type value",
			gin.H{"allowed_values": allowedFingerTypes}
	}

	if req.Hand == "" {
		return http.StatusBadRequest, "hand is required",
			gin.H{"allowed_values": allowedHands}
	}
	if !validHands[req.Hand] {
		return http.StatusBadRequest, "Invalid hand value",
			gin.H{"allowed_values": allowedHands}
	}

	if req.AttemptCount <= 0 {
		return http.StatusUnprocessableEntity, "attempt_count must be greater than 0", nil
	}
	if req.Nfiq2Score < 0 || req.Nfiq2Score > 100 {
		return http.StatusUnprocessableEntity, "nfiq2_score must be between 0 and 100", nil
	}
	if req.BlurScore < 0 || req.BlurScore > 1 {
		return http.StatusUnprocessableEntity, "blur_score must be between 0.0 and 1.0", nil
	}
	if req.BrightnessScore < 0 || req.BrightnessScore > 1 {
		return http.StatusUnprocessableEntity, "brightness_score must be between 0.0 and 1.0", nil
	}
	if req.GlareScore < 0 || req.GlareScore > 1 {
		return http.StatusUnprocessableEntity, "glare_score must be between 0.0 and 1.0", nil
	}

	return 0, "", nil
}

// captureValidationField returns a short field name hint for batch error reporting.
func captureValidationField(statusCode int, req model.CaptureRequest) string {
	switch {
	case req.SessionID == "":
		return "session_id"
	case req.ResidentPseudonymID == "":
		return "resident_pseudonym_id"
	case req.OperatorID == "":
		return "operator_id"
	case req.FingerType == "" || !validFingerTypes[req.FingerType]:
		return "finger_type"
	case req.Hand == "" || !validHands[req.Hand]:
		return "hand"
	case req.AttemptCount <= 0:
		return "attempt_count"
	default:
		return "score"
	}
}

// mapCaptureError writes the correct error response for service-layer errors.
func mapCaptureError(ctx *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		respondError(ctx, http.StatusNotFound, "Session not found")
		return
	}
	if errors.Is(err, repository.ErrDuplicateCapture) {
		respondError(ctx, http.StatusConflict, "This finger has already been captured for this session")
		return
	}
	var fkErr *repository.ErrForeignKeyViolation
	if errors.As(err, &fkErr) {
		respondError(ctx, http.StatusNotFound, "Referenced "+fkErr.Field+" does not exist")
		return
	}
	log.Printf("Capture upload error: %v", err)
	respondError(ctx, http.StatusInternalServerError, "An unexpected error occurred")
}

// readAll reads all bytes from a reader into a slice.
func readAll(r interface{ Read([]byte) (int, error) }) []byte {
	out := make([]byte, 0)
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return out
}

// getFormValue safely reads a form value from multipart form map
func getFormValue(values map[string][]string, key string) string {
	if vals, ok := values[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}
