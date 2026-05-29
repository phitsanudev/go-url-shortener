package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bell/go-url-shortener/backend/internal/application"
	"github.com/go-chi/chi/v5"
)

type URLHandler struct {
	service *application.URLService
}

type shortenRequest struct {
	URL string `json:"url"`
}

type errorResponse struct {
	Message string `json:"message"`
}

func NewURLHandler(service *application.URLService) *URLHandler {
	return &URLHandler{service: service}
}

func (h *URLHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *URLHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest

	// 1. แกะข้อมูล JSON Request Body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 2. เรียกใช้งาน Service ในการย่อลิงก์
	result, err := h.service.Shorten(r.Context(), req.URL)
	if err != nil {
		if errors.Is(err, application.ErrInvalidURL) {
			writeError(w, http.StatusBadRequest, "url must start with http:// or https://")
			return
		}
		writeError(w, http.StatusInternalServerError, "cannot shorten url")
		return
	}

	// 3. ส่งข้อมูลผลลัพธ์กลับในรูปแบบ JSON (201 Created)
	writeJSON(w, http.StatusCreated, result)
}

// Redirect ทำหน้าที่รับรหัสย่อ ดึง URL จริง แล้ว Redirect ไปยังหน้าเว็บจริง
func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	// 1. ดึงรหัสย่อจาก URL Path Parameter (ใช้ Router Chi)
	code := chi.URLParam(r, "code")

	// 2. ค้นหา URL จริงจากรหัสย่อ
	originalURL, err := h.service.Resolve(r.Context(), code)
	if err != nil {
		// ถ้าไม่พบ หรือ หมดอายุ ให้ส่งหน้า 404
		if errors.Is(err, application.ErrNotFound) || errors.Is(err, application.ErrExpired) {
			writeError(w, http.StatusNotFound, "short url not found or expired")
			return
		}
		writeError(w, http.StatusInternalServerError, "cannot resolve short url")
		return
	}

	// 3. Redirect ผู้ใช้ไปยัง URL จริงด้วย Status 302 Found
	http.Redirect(w, r, originalURL, http.StatusFound)
}

// CleanupExpired เรียกใช้ฟังก์ชันลบลิงก์ที่หมดอายุแล้วออกจากระบบหลัก
func (h *URLHandler) CleanupExpired(w http.ResponseWriter, r *http.Request) {
	if err := h.service.CleanupExpired(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "cleanup failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "expired urls cleaned",
	})
}

// writeJSON ช่วยในการเขียนข้อมูลกลับเป็น JSON และตั้งค่า Header Content-Type
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError ช่วยจัดรูปแบบ Error Response ให้เป็นมาตรฐานเดียวกัน
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Message: message})
}
