package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/database"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/ratelimit"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

func TestAttendanceTerminalRoutesRegistered(t *testing.T) {
	router := (&Server{}).Router()
	wanted := map[string]bool{
		"POST /api/v1/attendance-terminal/attendance":                            false,
		"POST /api/v1/attendance-terminal/attendance/batch":                      false,
		"POST /api/v1/attendance-terminal/heartbeat":                             false,
		"GET /api/v1/attendance-terminal/enrollment-session":                     false,
		"GET /api/v1/attendance-terminals":                                       false,
		"GET /api/v1/attendance-terminals/live":                                  false,
		"GET /api/v1/attendance-terminals/:id/mappings":                          false,
		"POST /api/v1/attendance-terminals/:id/enrollment-sessions":              false,
		"DELETE /api/v1/attendance-terminals/:id/enrollment-sessions/:sessionId": false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, exists := wanted[key]; exists {
			wanted[key] = true
		}
	}
	for route, found := range wanted {
		if !found {
			t.Errorf("expected route %s to be registered", route)
		}
	}
}

func TestNormalizeAttendanceScanRequest(t *testing.T) {
	slot := 12
	tests := []struct {
		name    string
		request attendanceScanRequest
		wantErr bool
		result  string
	}{
		{
			name:    "matched scan without explicit result",
			request: attendanceScanRequest{IdempotencyKey: "scan-key-0001", FingerprintSlot: &slot},
			result:  "match",
		},
		{
			name:    "unmatched scan",
			request: attendanceScanRequest{IdempotencyKey: "scan-key-0002", Result: "NO_MATCH"},
			result:  "no_match",
		},
		{
			name:    "short idempotency key",
			request: attendanceScanRequest{IdempotencyKey: "short", FingerprintSlot: &slot},
			wantErr: true,
		},
		{
			name:    "slot out of range",
			request: attendanceScanRequest{IdempotencyKey: "scan-key-0003", FingerprintSlot: intPointerForTest(1001)},
			wantErr: true,
		},
		{
			name:    "no match must not carry a slot",
			request: attendanceScanRequest{IdempotencyKey: "scan-key-0004", Result: "no_match", FingerprintSlot: &slot},
			wantErr: true,
		},
		{
			name:    "unknown result",
			request: attendanceScanRequest{IdempotencyKey: "scan-key-0005", Result: "maybe"},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := test.request
			err := normalizeAttendanceScanRequest(&request)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if request.Result != test.result {
				t.Fatalf("result = %q, want %q", request.Result, test.result)
			}
		})
	}
}

func TestAttendanceSlotCapacity(t *testing.T) {
	if attendanceSlotCapacity != 1000 {
		t.Fatalf("attendance slot capacity = %d, want 1000", attendanceSlotCapacity)
	}
	for _, slot := range []int{1, 127, 128, 1000} {
		if !validateAttendanceSlot(slot) {
			t.Fatalf("slot %d should be valid for attendance", slot)
		}
	}
	for _, slot := range []int{0, -1, 1001} {
		if validateAttendanceSlot(slot) {
			t.Fatalf("slot %d should be invalid for attendance", slot)
		}
	}
}

func TestResolveAttendanceScanTimingUsesServerTimeWhenOnline(t *testing.T) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)

	// Online: the server receive time is authoritative, even if the device clock
	// drifted slightly.
	online := resolveAttendanceScanTiming(now.Add(-5*time.Second).Format(time.RFC3339), now)
	if online.Source != "server" || !online.OfficialAt.Equal(now) {
		t.Fatalf("online timing = %+v, want server authoritative time", online)
	}

	// A queued event synced after the outage keeps its capture time.
	late := resolveAttendanceScanTiming(now.Add(-10*time.Minute).Format(time.RFC3339), now)
	if late.Source != "device" || !late.SyncedLate || !late.OfficialAt.Equal(now.Add(-10*time.Minute)) {
		t.Fatalf("late timing = %+v, want capture time marked synced late", late)
	}

	// A device clock in the future must not move attendance into the future.
	skew := resolveAttendanceScanTiming(now.Add(30*time.Minute).Format(time.RFC3339), now)
	if skew.Source != "server_clock_skew" || !skew.OfficialAt.Equal(now) {
		t.Fatalf("skew timing = %+v, want server time", skew)
	}

	// Stale backlog is clamped to server time.
	old := resolveAttendanceScanTiming(now.Add(-attendanceMaxBacklogAge-time.Minute).Format(time.RFC3339), now)
	if old.Source != "server_too_old" || !old.OfficialAt.Equal(now) {
		t.Fatalf("old timing = %+v, want server time", old)
	}
}

func intPointerForTest(value int) *int {
	return &value
}

type attendanceTestEnv struct {
	db       *gorm.DB
	router   http.Handler
	security *security.Manager
}

func newAttendanceTestEnv(t *testing.T) *attendanceTestEnv {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	cfg := config.Config{
		AppEnv:           "test",
		DatabaseURL:      dsn,
		JWTSecret:        strings.Repeat("j", 32),
		SessionSecret:    strings.Repeat("s", 32),
		DeviceHMACSecret: "attendance-integration-secret",
		SessionTTLMember: time.Hour,
		SessionTTLAdmin:  time.Hour,
	}
	handler, err := database.InitializeDB(&cfg)
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	t.Cleanup(func() { _ = handler.CloseDB() })
	if err := handler.Migrate(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	securityManager := security.New(cfg.JWTSecret, cfg.SessionSecret)
	server := &Server{
		DB:       handler.DB,
		Config:   cfg,
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Security: securityManager,
		Limiter:  ratelimit.New(),
	}
	return &attendanceTestEnv{db: handler.DB, router: server.Router(), security: securityManager}
}

func (env *attendanceTestEnv) request(t *testing.T, method, path, bearer, deviceKey string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	if deviceKey != "" {
		request.Header.Set("X-Device-Token", deviceKey)
	}
	recorder := httptest.NewRecorder()
	env.router.ServeHTTP(recorder, request)
	return recorder
}

func (env *attendanceTestEnv) createAdminSession(t *testing.T) string {
	t.Helper()
	suffix := time.Now().UnixNano()
	user := models.User{Email: fmt.Sprintf("attendance-admin-%d@example.com", suffix), Role: "admin", Active: true}
	if err := env.db.Create(&user).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	token := fmt.Sprintf("attendance-admin-session-%d", suffix)
	expires := time.Now().UTC().Add(time.Hour)
	session := models.Session{
		UserID:       user.ID,
		TokenDigest:  env.security.TokenDigest(token),
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    &expires,
	}
	if err := env.db.Create(&session).Error; err != nil {
		t.Fatalf("create admin session: %v", err)
	}
	return token
}

func (env *attendanceTestEnv) createMember(t *testing.T) (models.Member, models.User) {
	t.Helper()
	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("attendance-member-%d@example.com", suffix)
	usn := fmt.Sprintf("4AL25AT%04d", suffix%10000)
	user := models.User{Email: email, Role: "member", Active: true}
	if err := env.db.Create(&user).Error; err != nil {
		t.Fatalf("create member user: %v", err)
	}
	member := models.Member{
		UserID:             user.ID,
		Name:               "Attendance Test Member",
		USN:                &usn,
		LoginID:            strings.ToLower(usn),
		AcademicYear:       3,
		Branch:             "CSE",
		Email:              email,
		Status:             "active",
		MustChangePassword: false,
	}
	if err := env.db.Create(&member).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}
	return member, user
}

func (env *attendanceTestEnv) createDevice(t *testing.T, key, uid, deviceType string) models.DeviceRegistration {
	t.Helper()
	device := models.DeviceRegistration{
		DeviceUID:    uid,
		Name:         "Attendance integration terminal",
		DeviceType:   deviceType,
		APIKeyDigest: hashDeviceAPIKey(key),
		Status:       "active",
	}
	if err := env.db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	return device
}

type attendanceEnrollmentResponse struct {
	Success    bool `json:"success"`
	Enrollment struct {
		SessionUID      string `json:"session_uid"`
		Status          string `json:"status"`
		Operation       string `json:"operation"`
		FingerprintSlot *int   `json:"fingerprint_slot"`
	} `json:"enrollment"`
	Mapping *struct {
		FingerprintSlot *int `json:"fingerprint_slot"`
	} `json:"mapping"`
}

type attendanceScanResponse struct {
	Success         bool   `json:"success"`
	Outcome         string `json:"outcome"`
	Recorded        bool   `json:"recorded"`
	Duplicate       bool   `json:"duplicate"`
	EventID         int64  `json:"event_id"`
	AttendanceLogID int64  `json:"attendance_log_id"`
	Message         string `json:"message"`
}

type attendanceLiveResponse struct {
	Success bool `json:"success"`
	Events  []struct {
		ID      int64  `json:"id"`
		Outcome string `json:"outcome"`
		Member  *struct {
			Name    string `json:"name"`
			LoginID string `json:"login_id"`
		} `json:"member"`
	} `json:"events"`
}

func TestAttendanceTerminalScanLifecycleIntegration(t *testing.T) {
	env := newAttendanceTestEnv(t)
	suffix := time.Now().UnixNano()
	terminalKey := fmt.Sprintf("attendance-terminal-key-%d", suffix)
	terminal := env.createDevice(t, terminalKey, fmt.Sprintf("attendance-%d", suffix), attendanceDeviceType)
	adminToken := env.createAdminSession(t)
	member, _ := env.createMember(t)

	// Heartbeat is accepted only from an attendance terminal and is visible to
	// the admin console.
	heartbeat := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/heartbeat", "", terminalKey, gin.H{
		"firmware_version": "1.0.0-test",
		"reader_model":     "R307S",
		"reader_connected": true,
		"reader_readiness": "ready",
		"queue_depth":      0,
	})
	if heartbeat.Code != http.StatusOK {
		t.Fatalf("heartbeat status = %d: %s", heartbeat.Code, heartbeat.Body.String())
	}
	terminals := env.request(t, http.MethodGet, "/api/v1/attendance-terminals", adminToken, "", nil)
	if terminals.Code != http.StatusOK {
		t.Fatalf("terminal list status = %d: %s", terminals.Code, terminals.Body.String())
	}
	if !strings.Contains(terminals.Body.String(), "R307S") {
		t.Fatalf("terminal list does not contain heartbeat state: %s", terminals.Body.String())
	}

	// Admin starts device-specific enrollment for the existing member.
	create := env.request(t, http.MethodPost, fmt.Sprintf("/api/v1/attendance-terminals/%d/enrollment-sessions", terminal.ID), adminToken, "", gin.H{
		"member_id": member.ID,
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create enrollment status = %d: %s", create.Code, create.Body.String())
	}
	var enrollment attendanceEnrollmentResponse
	if err := json.Unmarshal(create.Body.Bytes(), &enrollment); err != nil {
		t.Fatalf("decode enrollment: %v", err)
	}
	if enrollment.Enrollment.SessionUID == "" || enrollment.Enrollment.FingerprintSlot == nil {
		t.Fatalf("enrollment missing session or slot: %s", create.Body.String())
	}
	slot := *enrollment.Enrollment.FingerprintSlot
	sessionPath := fmt.Sprintf("/api/v1/attendance-terminal/enrollment-sessions/%s", enrollment.Enrollment.SessionUID)

	// Device claims and completes the command. The template stays on the reader;
	// the backend only persists the slot mapping.
	poll := env.request(t, http.MethodGet, "/api/v1/attendance-terminal/enrollment-session", "", terminalKey, nil)
	if poll.Code != http.StatusOK || !strings.Contains(poll.Body.String(), `"action":"enroll"`) {
		t.Fatalf("enrollment poll = %d: %s", poll.Code, poll.Body.String())
	}
	if claim := env.request(t, http.MethodPost, sessionPath+"/claim", "", terminalKey, gin.H{}); claim.Code != http.StatusOK {
		t.Fatalf("claim status = %d: %s", claim.Code, claim.Body.String())
	}
	complete := env.request(t, http.MethodPost, sessionPath+"/complete", "", terminalKey, gin.H{
		"fingerprint_slot": slot,
		"device_user_id":   member.LoginID,
	})
	if complete.Code != http.StatusOK {
		t.Fatalf("complete status = %d: %s", complete.Code, complete.Body.String())
	}
	var mappings []models.BiometricEnrollment
	if err := env.db.Where("device_registration_id = ? AND member_id = ?", terminal.ID, member.ID).Find(&mappings).Error; err != nil {
		t.Fatalf("load mappings: %v", err)
	}
	if len(mappings) != 1 || mappings[0].FingerprintSlot == nil || *mappings[0].FingerprintSlot != slot {
		t.Fatalf("unexpected mappings: %+v", mappings)
	}
	if mappings[0].ReaderKey == nil || *mappings[0].ReaderKey != attendanceReaderKey {
		t.Fatalf("mapping must be scoped to the attendance reader: %+v", mappings[0])
	}

	scanBody := gin.H{"idempotency_key": fmt.Sprintf("scan-%d", suffix), "fingerprint_slot": slot}
	first := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/attendance", "", terminalKey, scanBody)
	if first.Code != http.StatusOK {
		t.Fatalf("first scan = %d: %s", first.Code, first.Body.String())
	}
	var firstResult attendanceScanResponse
	_ = json.Unmarshal(first.Body.Bytes(), &firstResult)
	if firstResult.Outcome != attendanceOutcomeEntry || !firstResult.Recorded || firstResult.EventID == 0 {
		t.Fatalf("unexpected first scan result: %s", first.Body.String())
	}

	// Same idempotency key: acknowledged, no second row.
	retry := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/attendance", "", terminalKey, scanBody)
	var retryResult attendanceScanResponse
	_ = json.Unmarshal(retry.Body.Bytes(), &retryResult)
	if retry.Code != http.StatusOK || retryResult.EventID != firstResult.EventID || retryResult.Outcome != attendanceOutcomeEntry {
		t.Fatalf("idempotent retry = %d: %s", retry.Code, retry.Body.String())
	}

	// New key but immediate double scan: duplicate window, still one row.
	second := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/attendance", "", terminalKey, gin.H{
		"idempotency_key":  fmt.Sprintf("scan-dup-%d", suffix),
		"fingerprint_slot": slot,
	})
	var secondResult attendanceScanResponse
	_ = json.Unmarshal(second.Body.Bytes(), &secondResult)
	if secondResult.Outcome != attendanceOutcomeDuplicate || secondResult.Recorded || !secondResult.Duplicate {
		t.Fatalf("expected duplicate scan, got %s", second.Body.String())
	}
	var logCount int64
	if err := env.db.Model(&models.AttendanceLog{}).Where("member_id = ?", member.ID).Count(&logCount).Error; err != nil {
		t.Fatalf("count attendance logs: %v", err)
	}
	if logCount != 1 {
		t.Fatalf("expected exactly one attendance log, got %d", logCount)
	}

	// Unknown slot and unmatched finger never create attendance.
	for name, body := range map[string]gin.H{
		"unknown slot": {"idempotency_key": fmt.Sprintf("scan-unknown-%d", suffix), "fingerprint_slot": 99},
		"no match":     {"idempotency_key": fmt.Sprintf("scan-nomatch-%d", suffix), "result": "no_match"},
	} {
		recorder := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/attendance", "", terminalKey, body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d: %s", name, recorder.Code, recorder.Body.String())
		}
		var result attendanceScanResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &result)
		if result.Recorded {
			t.Fatalf("%s must not record attendance: %s", name, recorder.Body.String())
		}
	}

	// Live feed is admin-only and carries the member identity.
	live := env.request(t, http.MethodGet, "/api/v1/attendance-terminals/live", adminToken, "", nil)
	if live.Code != http.StatusOK {
		t.Fatalf("live feed status = %d: %s", live.Code, live.Body.String())
	}
	var liveResult attendanceLiveResponse
	if err := json.Unmarshal(live.Body.Bytes(), &liveResult); err != nil {
		t.Fatalf("decode live feed: %v", err)
	}
	entryFound := false
	for _, event := range liveResult.Events {
		if event.ID == firstResult.EventID && event.Outcome == attendanceOutcomeEntry {
			entryFound = true
			if event.Member == nil || event.Member.Name != member.Name {
				t.Fatalf("live event is missing member identity: %+v", event)
			}
		}
	}
	if !entryFound {
		t.Fatalf("live feed did not include the entry event: %s", live.Body.String())
	}

	// A disabled member is rejected at scan time.
	if err := env.db.Model(&models.Member{}).Where("id = ?", member.ID).Update("status", "inactive").Error; err != nil {
		t.Fatalf("disable member: %v", err)
	}
	disabled := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/attendance", "", terminalKey, gin.H{
		"idempotency_key":  fmt.Sprintf("scan-disabled-%d", suffix),
		"fingerprint_slot": slot,
	})
	var disabledResult attendanceScanResponse
	_ = json.Unmarshal(disabled.Body.Bytes(), &disabledResult)
	if disabledResult.Outcome != attendanceOutcomeMemberInactive || disabledResult.Recorded {
		t.Fatalf("expected inactive member rejection, got %s", disabled.Body.String())
	}

	// A door-lock device cannot use the attendance terminal API.
	doorKey := fmt.Sprintf("door-key-%d", suffix)
	env.createDevice(t, doorKey, fmt.Sprintf("door-%d", suffix), "esp32")
	doorScan := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/attendance", "", doorKey, gin.H{
		"idempotency_key":  fmt.Sprintf("door-scan-%d", suffix),
		"fingerprint_slot": slot,
	})
	if doorScan.Code != http.StatusForbidden {
		t.Fatalf("door device scan status = %d, want 403: %s", doorScan.Code, doorScan.Body.String())
	}

	// An idempotency key is bound to one device.
	otherKey := fmt.Sprintf("attendance-other-%d", suffix)
	env.createDevice(t, otherKey, fmt.Sprintf("attendance-other-%d", suffix), attendanceDeviceType)
	conflict := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/attendance", "", otherKey, scanBody)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("cross-device idempotency status = %d, want 409: %s", conflict.Code, conflict.Body.String())
	}

	// Unauthenticated device calls are rejected.
	unauthorized := env.request(t, http.MethodPost, "/api/v1/attendance-terminal/attendance", "", "", gin.H{
		"idempotency_key":  fmt.Sprintf("anonymous-%d", suffix),
		"fingerprint_slot": slot,
	})
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous scan status = %d, want 401", unauthorized.Code)
	}
}

func TestAttendanceEnrollmentActivatesPendingMemberIntegration(t *testing.T) {
	env := newAttendanceTestEnv(t)
	suffix := time.Now().UnixNano()
	terminalKey := fmt.Sprintf("attendance-pending-key-%d", suffix)
	terminal := env.createDevice(t, terminalKey, fmt.Sprintf("attendance-pending-%d", suffix), attendanceDeviceType)
	adminToken := env.createAdminSession(t)

	loginID := fmt.Sprintf("pending-%d", suffix)
	usn := fmt.Sprintf("PEND%d", suffix%10000000)
	user := models.User{Email: "", Role: "member", Active: false}
	if err := env.db.Create(&user).Error; err != nil {
		t.Fatalf("create pending user: %v", err)
	}
	section := "A"
	member := models.Member{
		UserID:             user.ID,
		Name:               "Pending Attendance Member",
		USN:                &usn,
		LoginID:            loginID,
		AcademicYear:       1,
		Branch:             "CSE",
		Section:            &section,
		Email:              "",
		Phone:              "",
		MustChangePassword: true,
		Status:             memberStatusPendingEnrollment,
	}
	if err := env.db.Create(&member).Error; err != nil {
		t.Fatalf("create pending member: %v", err)
	}
	memberSession := models.MemberEnrollmentSession{
		SessionUID: fmt.Sprintf("pending-details-%d", suffix),
		MemberID:   member.ID,
		Status:     enrollmentStatusPendingDetails,
		IsActive:   true,
	}
	if err := env.db.Create(&memberSession).Error; err != nil {
		t.Fatalf("create pending details session: %v", err)
	}

	create := env.request(t, http.MethodPost, fmt.Sprintf("/api/v1/attendance-terminals/%d/enrollment-sessions", terminal.ID), adminToken, "", gin.H{
		"member_id": member.ID,
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create enrollment status = %d: %s", create.Code, create.Body.String())
	}
	var enrollment attendanceEnrollmentResponse
	if err := json.Unmarshal(create.Body.Bytes(), &enrollment); err != nil {
		t.Fatalf("decode enrollment: %v", err)
	}
	if enrollment.Enrollment.SessionUID == "" || enrollment.Enrollment.FingerprintSlot == nil {
		t.Fatalf("enrollment missing session or slot: %s", create.Body.String())
	}
	slot := *enrollment.Enrollment.FingerprintSlot
	sessionPath := fmt.Sprintf("/api/v1/attendance-terminal/enrollment-sessions/%s", enrollment.Enrollment.SessionUID)
	if claim := env.request(t, http.MethodPost, sessionPath+"/claim", "", terminalKey, gin.H{}); claim.Code != http.StatusOK {
		t.Fatalf("claim status = %d: %s", claim.Code, claim.Body.String())
	}
	complete := env.request(t, http.MethodPost, sessionPath+"/complete", "", terminalKey, gin.H{
		"fingerprint_slot": slot,
	})
	if complete.Code != http.StatusOK {
		t.Fatalf("complete status = %d: %s", complete.Code, complete.Body.String())
	}

	var updatedMember models.Member
	if err := env.db.First(&updatedMember, member.ID).Error; err != nil {
		t.Fatalf("reload member: %v", err)
	}
	if updatedMember.Status != "active" {
		t.Fatalf("member status = %q, want active", updatedMember.Status)
	}
	var updatedUser models.User
	if err := env.db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !updatedUser.Active {
		t.Fatal("user should be active after attendance enrollment")
	}
	var finalized models.MemberEnrollmentSession
	if err := env.db.First(&finalized, memberSession.ID).Error; err != nil {
		t.Fatalf("reload member enrollment session: %v", err)
	}
	if finalized.Status != enrollmentStatusFinalized || finalized.IsActive {
		t.Fatalf("member enrollment session = %+v, want finalized and inactive", finalized)
	}
	var mapping models.BiometricEnrollment
	if err := env.db.Where("member_id = ? AND device_registration_id = ?", member.ID, terminal.ID).First(&mapping).Error; err != nil {
		t.Fatalf("load mapping: %v", err)
	}
	if mapping.FingerprintSlot == nil || *mapping.FingerprintSlot != slot {
		t.Fatalf("mapping slot = %v, want %d", mapping.FingerprintSlot, slot)
	}
}
