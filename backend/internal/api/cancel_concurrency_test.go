package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func TestCancelEnrollmentWhilePollingConcurrent(t *testing.T) {
	env := setupEnrollmentTestEnv(t)
	memberID, _, _ := env.createPendingMember("enroll-cancel-race")
	session := env.startEnrollment(memberID, "login")
	slot := 21
	env.progress(session.SessionUID, enrollmentStatusVerifying, &slot, "", "")

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				recorder := env.request(http.MethodGet, fmt.Sprintf("/api/v1/members-management/%d/fingerprint-enrollment", memberID), env.adminToken, "", nil)
				if recorder.Code != http.StatusOK && recorder.Code != http.StatusNotFound {
					t.Errorf("status during poll = %d: %s", recorder.Code, recorder.Body.String())
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	time.Sleep(20 * time.Millisecond)
	start := time.Now()
	recorder := env.request(http.MethodDelete, fmt.Sprintf("/api/v1/members-management/%d/fingerprint-enrollment", memberID), env.adminToken, "", nil)
	elapsed := time.Since(start)
	close(stop)
	wg.Wait()

	if elapsed > 2*time.Second {
		t.Fatalf("cancel took %s while status polling was active", elapsed)
	}
	env.expect(recorder, http.StatusOK)
}

func TestCancelEnrollmentConcurrentRequests(t *testing.T) {
	env := setupEnrollmentTestEnv(t)
	memberID, _, _ := env.createPendingMember("enroll-cancel-double")
	session := env.startEnrollment(memberID, "login")
	slot := 22
	env.progress(session.SessionUID, enrollmentStatusVerifying, &slot, "", "")

	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- env.request(http.MethodDelete, fmt.Sprintf("/api/v1/members-management/%d/fingerprint-enrollment", memberID), env.adminToken, "", nil)
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	for recorder := range results {
		if recorder.Code != http.StatusOK {
			t.Fatalf("double cancel status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
	}

	var memberCount int64
	env.db.Model(&models.Member{}).Where("id = ?", memberID).Count(&memberCount)
	if memberCount != 0 {
		t.Fatal("double cancellation left the pending member in place")
	}
}

func TestCancelEnrollmentStaysIdempotentWhileDeviceOffline(t *testing.T) {
	env := setupEnrollmentTestEnv(t)
	memberID, _, _ := env.createPendingMember("enroll-cancel-offline")
	member := env.loadMember(memberID)
	session := env.startEnrollment(memberID, "login")
	slot := 23
	env.progress(session.SessionUID, enrollmentStatusVerifying, &slot, "", "")

	start := time.Now()
	recorder := env.request(http.MethodDelete, fmt.Sprintf("/api/v1/members-management/%d/fingerprint-enrollment", memberID), env.adminToken, "", nil)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("cancel waited %s for an offline reader", elapsed)
	}
	env.expect(recorder, http.StatusOK)
	var first enrollmentTestResponse
	env.decode(recorder, &first)
	if first.Enrollment == nil || first.Enrollment.Status != enrollmentStatusCancelled || !first.Enrollment.CleanupRequired {
		t.Fatalf("unexpected cancel response: %s", recorder.Body.String())
	}

	recorder = env.request(http.MethodDelete, fmt.Sprintf("/api/v1/members-management/%d/fingerprint-enrollment", memberID), env.adminToken, "", nil)
	env.expect(recorder, http.StatusOK)
	var repeated enrollmentTestResponse
	env.decode(recorder, &repeated)
	if repeated.Enrollment == nil || repeated.Enrollment.SessionUID != session.SessionUID || !repeated.Enrollment.CleanupRequired {
		t.Fatalf("unexpected repeated cancel response: %s", recorder.Body.String())
	}

	var memberCount int64
	env.db.Model(&models.Member{}).Where("id = ?", memberID).Count(&memberCount)
	if memberCount != 0 {
		t.Fatal("cancelled pending member was not deleted")
	}
	var userCount int64
	env.db.Model(&models.User{}).Where("id = ?", member.UserID).Count(&userCount)
	if userCount != 0 {
		t.Fatal("cancelled pending user was not deleted")
	}
	var cleanupSession models.MemberEnrollmentSession
	if err := env.db.Where("session_uid = ?", session.SessionUID).First(&cleanupSession).Error; err != nil {
		t.Fatalf("load cleanup session: %v", err)
	}
	if cleanupSession.Status != enrollmentStatusCancelled || !cleanupSession.CleanupRequired || cleanupSession.FingerprintSlot == nil || *cleanupSession.FingerprintSlot != slot {
		t.Fatalf("cleanup was not preserved for the offline reader: %+v", cleanupSession)
	}

	claim := env.claim()
	if claim.Action != "cleanup" || claim.Enrollment == nil || claim.Enrollment.SessionUID != session.SessionUID {
		t.Fatalf("unexpected offline cleanup claim: %+v", claim)
	}
	recorder = env.request(http.MethodPost, fmt.Sprintf("/api/v1/biometric/enrollment-sessions/%s/cleanup", session.SessionUID), "", env.deviceKey, map[string]any{"fingerprint_slot": slot})
	env.expect(recorder, http.StatusOK)
	if claim := env.claim(); claim.Action != "none" {
		t.Fatalf("cleanup remained pending after acknowledgment: %+v", claim)
	}
}
