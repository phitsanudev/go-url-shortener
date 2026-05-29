package application

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/bell/go-url-shortener/backend/internal/domain"
)

// Struct สำหรับจัดการ Logic ทั้งหมดของการย่อลิงก์และการดึงลิงก์เดิมกลับมา
type URLService struct {
	repo    URLRepository
	cache   URLCache
	baseURL string
	ttl     time.Duration
}

// Struct สำหรับเตรียมข้อมูลที่จะส่งกลับหลังการย่อลิงก์สำเร็จ
type ShortenResult struct {
	ShortCode string `json:"shortCode"`
	ShortURL  string `json:"shortUrl"`
	ExpiresAt string `json:"expiresAt"`
}

func NewURLService(repo URLRepository, cache URLCache, baseURL string, ttl time.Duration) *URLService {
	return &URLService{
		repo:    repo,
		cache:   cache,
		baseURL: strings.TrimRight(baseURL, "/"),
		ttl:     ttl,
	}
}

// Shorten ทำหน้าที่ย่อ URL ยาวๆ ให้สั้นลง
func (s *URLService) Shorten(ctx context.Context, rawURL string) (*ShortenResult, error) {
	// 1. ตรวจสอบก่อนว่า URL อยู่ในรูปแบบที่ถูกต้องหรือไม่
	if !isValidURL(rawURL) {
		return nil, ErrInvalidURL
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)

	var code string
	var err error

	// 2. ลองสุ่มรหัสย่อ (Short Code) สูงสุด 5 ครั้ง เผื่อกรณีสุ่มรหัสแล้วไปชนกับข้อมูลเดิมในระบบ
	for i := 0; i < 5; i++ {
		code, err = generateShortCode(7)
		if err != nil {
			return nil, err
		}

		item := domain.ShortURL{
			ShortCode:   code,
			OriginalURL: rawURL,
			CreatedAt:   now,
			ExpiresAt:   expiresAt,
		}

		// 3. บันทึกรหัสย่อลง Database PostgreSQL
		err = s.repo.Save(ctx, item)
		if err == nil {
			// บันทึกลง Database สำเร็จ ให้บันทึกข้อมูลลง Cache ใน Redis ด้วย
			_ = s.cache.Set(ctx, code, rawURL, s.ttl)
			return &ShortenResult{
				ShortCode: code,
				ShortURL:  s.baseURL + "/" + code,
				ExpiresAt: expiresAt.Format(time.RFC3339),
			}, nil
		}
	}

	// ถ้าสุ่มครบ 5 ครั้งแล้วยังเกิดการชนอยู่ Return Err
	return nil, err
}

// Resolve ทำหน้าที่ค้นหา URL จริง จากรหัสย่อ (Short Code) ที่รับเข้ามา
func (s *URLService) Resolve(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", ErrNotFound
	}

	// 1. ดึงข้อมูลจาก Cache ก่อนเพื่อความรวดเร็ว (Cache Hit)
	cachedURL, err := s.cache.Get(ctx, code)
	if err == nil && cachedURL != "" {
		return cachedURL, nil
	}

	// 2. ถ้าไม่เจอใน Cache (Cache Miss) ให้ไปดึงข้อมูลจาก Database
	item, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}

	// 3. ตรวจสอบว่าลิงก์หมดอายุหรือยัง
	now := time.Now().UTC()
	if item.IsExpired(now) {
		// ถ้าหมดอายุแล้ว ให้ลบข้อมูลออกจาก Cache และคืนค่า Error
		_ = s.cache.Delete(ctx, code)
		return "", ErrExpired
	}

	// 4. บันทึกข้อมูลที่เพิ่งสืบค้นได้กลับลงใน Cache พร้อมตั้งระยะเวลาหมดอายุตามเวลาที่เหลืออยู่
	ttlRemaining := time.Until(item.ExpiresAt)
	if ttlRemaining > 0 {
		_ = s.cache.Set(ctx, code, item.OriginalURL, ttlRemaining)
	}

	return item.OriginalURL, nil
}

// CleanupExpired ลบข้อมูลลิงก์ที่หมดอายุแล้วทั้งหมดออกจากฐานข้อมูลหลัก
func (s *URLService) CleanupExpired(ctx context.Context) error {
	return s.repo.DeleteExpired(ctx)
}

// isValidURL ตรวจสอบรูปแบบ URL ว่าขึ้นต้นด้วย http:// หรือ https:// และมี Host หรือไม่
func isValidURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	return parsed.Host != ""
}
