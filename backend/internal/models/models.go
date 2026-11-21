package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// User represents a system user with role-based access
type User struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	Email          string    `gorm:"not null;uniqueIndex" json:"email"`
	Phone          *string   `json:"phone,omitempty"`
	PasswordDigest *string   `json:"-"` // Never expose password hash in JSON
	Role           string    `gorm:"default:'member';index" json:"role"`
	Active         bool      `gorm:"default:true" json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relations
	Member   *Member   `gorm:"foreignKey:UserID" json:"member,omitempty"`
	Admin    *Admin    `gorm:"foreignKey:UserID" json:"admin,omitempty"`
	Sessions []Session `gorm:"foreignKey:UserID" json:"-"`
}

// Member represents a student/lab member
type Member struct {
	ID                 int64          `gorm:"primaryKey" json:"id"`
	UserID             int64          `json:"user_id"`
	Name               string         `gorm:"not null" json:"name"`
	USN                *string        `json:"usn,omitempty"` // University Seat Number
	RollNumber         *string        `json:"roll_number,omitempty"`
	LoginID            string         `gorm:"not null;uniqueIndex" json:"login_id"`
	AcademicYear       int            `json:"academic_year"`
	Branch             string         `gorm:"not null" json:"branch"`
	Section            *string        `json:"section,omitempty"`
	Email              string         `gorm:"not null" json:"email"`
	Phone              string         `json:"phone,omitempty"`
	PhoneVerified      bool           `gorm:"default:false" json:"phone_verified"`
	MustChangePassword bool           `gorm:"default:true" json:"must_change_password"`
	LoginIDChanged     bool           `gorm:"default:false" json:"login_id_changed"`
	LoginIDChangedAt   *time.Time     `json:"login_id_changed_at,omitempty"`
	Status             string         `gorm:"default:'active'" json:"status"`
	GitHubURL          *string        `gorm:"column:github_url" json:"github_url,omitempty"`
	LinkedInURL        *string        `gorm:"column:linkedin_url" json:"linkedin_url,omitempty"`
	PortfolioURL       *string        `json:"portfolio_url,omitempty"`
	Bio                *string        `json:"bio,omitempty"`
	Interests          pq.StringArray `json:"interests" gorm:"type:text[]"`
	DeletedAt          *time.Time     `json:"deleted_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`

	// EnrollmentStatus is populated by management serializers from the latest
	// member enrollment session. It is not persisted on the members table.
	EnrollmentStatus string `gorm:"-" json:"enrollment_status,omitempty"`

	// Relations
	User                   User                   `gorm:"foreignKey:UserID" json:"-"`
	AttendanceLogs         []AttendanceLog        `gorm:"foreignKey:MemberID" json:"-"`
	BorrowRecords          []BorrowRecord         `gorm:"foreignKey:MemberID" json:"-"`
	ExtensionRequests      []ExtensionRequest     `gorm:"foreignKey:MemberID" json:"-"`
	ReturnRequests         []ReturnRequest        `gorm:"foreignKey:MemberID" json:"-"`
	TaskSubmissionStatuses []TaskSubmissionStatus `gorm:"foreignKey:MemberID" json:"-"`
	GroupMessages          []GroupMessage         `gorm:"foreignKey:UserID" json:"-"`
}

// Admin represents a lab administrator
type Admin struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `gorm:"not null" json:"name"`
	IsPrimary bool      `gorm:"default:false" json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// Session represents an authenticated session
type Session struct {
	ID           int64      `gorm:"primaryKey" json:"-"`
	UserID       int64      `json:"-"`
	TokenDigest  string     `gorm:"not null;uniqueIndex" json:"-"`
	IPAddress    *string    `json:"ip_address,omitempty"`
	UserAgent    *string    `json:"user_agent,omitempty"`
	LastActiveAt time.Time  `json:"last_active_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// OAuthLoginHandoff is a short-lived, single-use code used to transfer an
// OAuth-authenticated login from the API callback to the web client.
type OAuthLoginHandoff struct {
	ID         int64      `gorm:"primaryKey" json:"-"`
	CodeDigest string     `gorm:"not null;uniqueIndex" json:"-"`
	UserID     int64      `gorm:"not null;index" json:"-"`
	ExpiresAt  time.Time  `gorm:"not null" json:"-"`
	UsedAt     *time.Time `json:"-"`
	CreatedAt  time.Time  `json:"-"`
}

func (OAuthLoginHandoff) TableName() string {
	return "oauth_login_handoffs"
}

type GmailOAuthCredential struct {
	ID                    int64     `gorm:"primaryKey" json:"-"`
	EncryptedRefreshToken string    `gorm:"not null" json:"-"`
	Scopes                string    `gorm:"not null;default:''" json:"-"`
	TokenType             string    `gorm:"not null;default:'Bearer'" json:"-"`
	AuthorizedAt          time.Time `gorm:"not null" json:"-"`
	CreatedAt             time.Time `json:"-"`
	UpdatedAt             time.Time `json:"-"`
}

func (GmailOAuthCredential) TableName() string {
	return "gmail_oauth_credentials"
}

// OTPCode represents a one-time password for verification
type OTPCode struct {
	ID             int64     `gorm:"primaryKey" json:"-"`
	UserID         int64     `json:"-"`
	OTPDigest      string    `gorm:"not null" json:"-"`
	DeliveryMethod string    `gorm:"not null" json:"delivery_method,omitempty"`
	Purpose        string    `gorm:"not null;index" json:"purpose,omitempty"` // login, verification, password_reset
	Attempts       int       `gorm:"default:0" json:"-"`
	Used           bool      `gorm:"default:false" json:"-"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BiometricEnrollment represents a member's biometric enrollment
type BiometricEnrollment struct {
	ID                   int64     `gorm:"primaryKey" json:"id"`
	MemberID             int64     `json:"member_id"`
	DeviceRegistrationID int64     `json:"device_registration_id"`
	ReaderKey            *string   `gorm:"size:32" json:"reader_key,omitempty"` // login, logout, dedicated; null for legacy mappings
	DeviceUserID         string    `json:"device_user_id"`                      // Device-internal template identifier
	FingerprintSlot      *int      `json:"fingerprint_slot,omitempty"`          // R307S template slot
	Status               string    `gorm:"default:'pending'" json:"status"`     // pending, enrolled, verified
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`

	// Relations
	Member             Member             `gorm:"foreignKey:MemberID" json:"-"`
	DeviceRegistration DeviceRegistration `gorm:"foreignKey:DeviceRegistrationID" json:"-"`
}

// MemberEnrollmentSession tracks the two-scan fingerprint enrollment that must
// finish before a newly created member account is activated.
type MemberEnrollmentSession struct {
	ID                   int64      `gorm:"primaryKey" json:"id"`
	SessionUID           string     `gorm:"not null;uniqueIndex" json:"session_uid"`
	MemberID             int64      `gorm:"index" json:"member_id"`
	RequestedByID        int64      `json:"requested_by_id"`
	DeviceRegistrationID *int64     `json:"device_registration_id,omitempty"`
	ReaderKey            *string    `gorm:"size:32" json:"reader_key,omitempty"`
	Status               string     `gorm:"not null;default:'pending_details';index" json:"status"`
	FingerprintSlot      *int       `json:"fingerprint_slot,omitempty"`
	DeviceUserID         *string    `json:"device_user_id,omitempty"`
	Attempts             int        `gorm:"not null;default:0" json:"attempts"`
	ErrorCode            *string    `json:"error_code,omitempty"`
	ErrorMessage         *string    `json:"error_message,omitempty"`
	CleanupRequired      bool       `gorm:"not null;default:false" json:"cleanup_required"`
	IsActive             bool       `gorm:"not null;default:true;index" json:"is_active"`
	ClaimedAt            *time.Time `json:"claimed_at,omitempty"`
	ClaimedByDeviceID    *int64     `json:"claimed_by_device_id,omitempty"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	CleanupCompletedAt   *time.Time `json:"cleanup_completed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// AttendanceEnrollmentSession tracks an enrollment or removal command for an
// attendance terminal. Fingerprint templates stay on the terminal; only the
// device/slot mapping is persisted in biometric_enrollments.
type AttendanceEnrollmentSession struct {
	ID                   int64      `gorm:"primaryKey" json:"id"`
	SessionUID           string     `gorm:"not null;uniqueIndex" json:"session_uid"`
	MemberID             int64      `gorm:"not null;index" json:"member_id"`
	DeviceRegistrationID int64      `gorm:"not null;index" json:"device_registration_id"`
	RequestedByID        int64      `gorm:"not null;default:0" json:"requested_by_id"`
	Operation            string     `gorm:"not null;default:'enroll';size:24" json:"operation"`
	Status               string     `gorm:"not null;default:'pending';index" json:"status"`
	ReaderKey            *string    `gorm:"size:32" json:"reader_key,omitempty"`
	FingerprintSlot      *int       `json:"fingerprint_slot,omitempty"`
	DeviceUserID         *string    `json:"device_user_id,omitempty"`
	Attempts             int        `gorm:"not null;default:0" json:"attempts"`
	ErrorCode            *string    `json:"error_code,omitempty"`
	ErrorMessage         *string    `json:"error_message,omitempty"`
	CleanupRequired      bool       `gorm:"not null;default:false" json:"cleanup_required"`
	CleanupCompletedAt   *time.Time `json:"cleanup_completed_at,omitempty"`
	IsActive             bool       `gorm:"not null;default:true;index" json:"is_active"`
	ClaimedAt            *time.Time `json:"claimed_at,omitempty"`
	ClaimedByDeviceID    *int64     `json:"claimed_by_device_id,omitempty"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// AttendanceTerminalStatus is the latest health report from an attendance
// terminal. It intentionally stores no biometric or member data.
type AttendanceTerminalStatus struct {
	ID                   int64      `gorm:"primaryKey" json:"id"`
	DeviceRegistrationID int64      `gorm:"not null;uniqueIndex" json:"device_registration_id"`
	FirmwareVersion      string     `gorm:"size:64" json:"firmware_version"`
	ReaderModel          string     `gorm:"size:64" json:"reader_model"`
	ReaderFirmware       string     `gorm:"size:64" json:"reader_firmware"`
	ReaderConnected      bool       `gorm:"not null;default:false" json:"reader_connected"`
	ReaderReadiness      string     `gorm:"not null;default:'not_ready';size:32" json:"reader_readiness"`
	QueueDepth           int        `gorm:"not null;default:0" json:"queue_depth"`
	NetworkStatus        string     `gorm:"size:32" json:"network_status"`
	LastSyncAt           *time.Time `json:"last_sync_at,omitempty"`
	LastCaptureAt        *time.Time `json:"last_capture_at,omitempty"`
	ReportedAt           *time.Time `json:"reported_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (AttendanceTerminalStatus) TableName() string {
	return "attendance_terminal_status"
}

type MemberRegistrationEmail struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	MemberID  int64      `gorm:"not null;uniqueIndex:index_member_registration_emails_on_member_id" json:"member_id"`
	Recipient string     `gorm:"not null" json:"recipient"`
	Status    string     `gorm:"not null;default:'pending';index" json:"status"`
	Attempts  int        `gorm:"not null;default:0" json:"attempts"`
	LastError *string    `json:"last_error,omitempty"`
	QueuedAt  *time.Time `json:"queued_at,omitempty"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// DeviceRegistration represents a fingerprint/biometric device registration
type DeviceRegistration struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	DeviceUID    string     `gorm:"not null;uniqueIndex" json:"device_uid"`
	Name         string     `gorm:"not null" json:"name"`
	DeviceType   string     `gorm:"not null" json:"device_type"` // r307s, esp32, etc.
	APIKeyDigest string     `gorm:"not null;uniqueIndex" json:"-"`
	Status       string     `gorm:"default:'active'" json:"status"` // active, inactive, revoked
	TokenVersion int        `gorm:"not null;default:0" json:"-"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relations
	BiometricEnrollments []BiometricEnrollment `gorm:"foreignKey:DeviceRegistrationID" json:"-"`
	Readers              []DeviceReader        `gorm:"foreignKey:DeviceRegistrationID" json:"-"`
}

// DeviceReader is a reader reported by an ESP32 controller. The backend only
// stores availability metadata; fingerprint templates remain on the sensor.
type DeviceReader struct {
	ID                   int64      `gorm:"primaryKey" json:"id"`
	DeviceRegistrationID int64      `gorm:"not null;uniqueIndex:idx_device_readers_device_key;index" json:"device_registration_id"`
	ReaderKey            string     `gorm:"not null;size:32;uniqueIndex:idx_device_readers_device_key" json:"reader_key"` // login, logout, dedicated
	Label                string     `gorm:"not null;size:64" json:"label"`
	UART                 int        `gorm:"not null;default:0" json:"uart"`
	Connected            bool       `gorm:"not null;default:false" json:"connected"`
	Readiness            string     `gorm:"not null;default:'not_ready';size:32" json:"readiness"`
	LastSeenAt           *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// DeviceEvent represents a fingerprint scan or device event
type DeviceEvent struct {
	ID                   int64           `gorm:"primaryKey" json:"id"`
	DeviceRegistrationID int64           `json:"device_registration_id"`
	EventType            string          `gorm:"not null" json:"event_type"` // check_in, check_out, enrollment, verification
	IdempotencyKey       string          `gorm:"not null;uniqueIndex" json:"idempotency_key"`
	MemberIdentifier     *string         `json:"member_identifier,omitempty"`
	HMACSignature        *string         `json:"-"` // Internal use only
	EventTimestamp       time.Time       `json:"event_timestamp"`
	Payload              json.RawMessage `json:"payload,omitempty" gorm:"type:jsonb"`
	ProcessingStatus     string          `gorm:"default:'pending'" json:"processing_status"` // pending, processed, failed
	ProcessingError      *string         `json:"processing_error,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`

	// Relations
	DeviceRegistration DeviceRegistration `gorm:"foreignKey:DeviceRegistrationID" json:"-"`
}

// AttendanceLog represents a member's attendance record
type AttendanceLog struct {
	ID              int64      `gorm:"primaryKey" json:"id"`
	MemberID        int64      `json:"member_id"`
	EntryDate       time.Time  `gorm:"type:date" json:"entry_date"`
	EntryTime       time.Time  `json:"entry_time"`
	ExitTime        *time.Time `json:"exit_time,omitempty"`
	Source          string     `gorm:"default:'manual'" json:"source"` // manual, device, web
	DeviceReference *string    `json:"device_reference,omitempty"`
	CreatedByID     *int64     `json:"created_by_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Relations
	Member Member `gorm:"foreignKey:MemberID" json:"-"`
}

// Announcement represents a lab announcement
type Announcement struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"not null" json:"title"`
	Content     string     `gorm:"type:text" json:"content"`
	IsPinned    bool       `gorm:"default:false" json:"is_pinned"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	CreatedByID int64      `json:"created_by_id"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Relations
	CreatedBy User               `gorm:"foreignKey:CreatedByID" json:"-"`
	Reads     []AnnouncementRead `gorm:"foreignKey:AnnouncementID" json:"-"`
}

// AnnouncementRead tracks which users have read an announcement
type AnnouncementRead struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	AnnouncementID int64     `json:"announcement_id"`
	UserID         int64     `json:"user_id"`
	ReadAt         time.Time `json:"read_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relations
	Announcement Announcement `gorm:"foreignKey:AnnouncementID" json:"-"`
	User         User         `gorm:"foreignKey:UserID" json:"-"`
}

// Notification represents a notification for a user
type Notification struct {
	ID               int64      `gorm:"primaryKey" json:"id"`
	UserID           int64      `json:"user_id"`
	Title            string     `gorm:"not null" json:"title"`
	Body             *string    `json:"body,omitempty"`
	NotificationType string     `gorm:"not null;index" json:"notification_type"` // system, announcement, borrowing, etc.
	SubjectType      *string    `json:"subject_type,omitempty"`                  // user, member, borrow_record, etc.
	SubjectID        *int64     `json:"subject_id,omitempty"`
	IsRead           bool       `gorm:"default:false" json:"is_read"`
	ReadAt           *time.Time `json:"read_at,omitempty"`
	EmailStatus      *string    `json:"email_status,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// InventoryItem represents an item in the lab inventory
type InventoryItem struct {
	ID                int64      `gorm:"primaryKey" json:"id"`
	Name              string     `gorm:"not null" json:"name"`
	Category          string     `gorm:"not null" json:"category"`
	Description       *string    `json:"description,omitempty"`
	TotalQuantity     int        `gorm:"default:0" json:"total_quantity"`
	AvailableQuantity int        `gorm:"default:0" json:"available_quantity"`
	StorageLocation   *string    `json:"storage_location,omitempty"`
	Condition         *string    `json:"condition,omitempty"`
	IsActive          bool       `gorm:"default:true" json:"is_active"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Relations
	BorrowRecords []BorrowRecord `gorm:"foreignKey:InventoryItemID" json:"-"`
}

// Semester represents an academic semester window.
type Semester struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	StartDate time.Time `gorm:"type:date" json:"start_date"`
	EndDate   time.Time `gorm:"type:date" json:"end_date"`
	IsCurrent bool      `gorm:"column:is_current;default:false" json:"is_current"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InventoryImport records a bulk inventory upload.
type InventoryImport struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Filename     string    `gorm:"not null" json:"filename"`
	TotalRows    int       `gorm:"not null" json:"total_rows"`
	ImportedRows int       `gorm:"default:0" json:"imported_rows"`
	ErrorRows    int       `gorm:"default:0" json:"error_rows"`
	Status       string    `gorm:"default:'completed'" json:"status"`
	ImportedByID int64     `gorm:"column:imported_by_id" json:"imported_by_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BorrowRecord represents a member borrowing an item
type BorrowRecord struct {
	ID                int64      `gorm:"primaryKey" json:"id"`
	MemberID          int64      `json:"member_id"`
	InventoryItemID   int64      `json:"inventory_item_id"`
	Status            string     `gorm:"default:'borrowed'" json:"status"` // borrowed, returned, overdue
	BorrowedAt        time.Time  `json:"borrowed_at"`
	DueDate           time.Time  `json:"due_date"`
	ReturnedAt        *time.Time `json:"returned_at,omitempty"`
	ConditionOnReturn *string    `json:"condition_on_return,omitempty"`
	IdempotencyKey    *string    `json:"-"`
	ApprovedByID      *int64     `json:"approved_by_id,omitempty"`
	ApprovalReason    *string    `json:"approval_reason,omitempty"`
	Quantity          int        `gorm:"default:1" json:"quantity"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Relations
	Member        Member            `gorm:"foreignKey:MemberID" json:"-"`
	InventoryItem InventoryItem     `gorm:"foreignKey:InventoryItemID" json:"-"`
	Extension     *ExtensionRequest `gorm:"foreignKey:BorrowRecordID" json:"extension_request,omitempty"`
	ReturnRequest *ReturnRequest    `gorm:"foreignKey:BorrowRecordID" json:"return_request,omitempty"`
}

// ExtensionRequest represents a borrowing extension request
type ExtensionRequest struct {
	ID                  int64     `gorm:"primaryKey" json:"id"`
	BorrowRecordID      int64     `json:"borrow_record_id"`
	MemberID            int64     `json:"member_id"`
	RequestedReturnDate time.Time `json:"requested_return_date"`
	Reason              string    `gorm:"not null" json:"reason"`
	Status              string    `gorm:"default:'pending'" json:"status"` // pending, approved, rejected
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	// Relations
	Member       Member       `gorm:"foreignKey:MemberID" json:"-"`
	BorrowRecord BorrowRecord `gorm:"foreignKey:BorrowRecordID" json:"-"`
}

// ReturnRequest represents a return request for a borrowed item
type ReturnRequest struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	BorrowRecordID int64      `json:"borrow_record_id"`
	MemberID       int64      `json:"member_id"`
	Status         string     `gorm:"default:'pending'" json:"status"` // pending, confirmed, rejected
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
	ConfirmedByID  *int64     `json:"confirmed_by_id,omitempty"`
	ConditionNote  *string    `json:"condition_note,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relations
	BorrowRecord BorrowRecord `gorm:"foreignKey:BorrowRecordID" json:"-"`
	Member       Member       `gorm:"foreignKey:MemberID" json:"-"`
}

// Task represents a lab task/assignment
type Task struct {
	ID                  int64      `gorm:"primaryKey" json:"id"`
	Title               string     `gorm:"not null" json:"title"`
	Description         *string    `json:"description,omitempty"`
	TaskType            string     `gorm:"not null" json:"task_type"` // assignment, project, practical
	PublishedAt         *time.Time `json:"published_at,omitempty"`
	Deadline            time.Time  `json:"due_date"`
	Instructions        *string    `json:"instructions,omitempty"`
	ExternalLink        *string    `json:"external_link,omitempty"`
	RequiredAttachments *string    `json:"required_attachments,omitempty"`
	CreatedByID         int64      `json:"created_by_id"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	// Relations
	CreatedBy              User                   `gorm:"foreignKey:CreatedByID" json:"-"`
	TaskTargets            []TaskTarget           `gorm:"foreignKey:TaskID" json:"-"`
	TaskSubmissionStatuses []TaskSubmissionStatus `gorm:"foreignKey:TaskID" json:"-"`
}

// TaskTarget represents task requirements by academic year
type TaskTarget struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	TaskID       int64     `json:"task_id"`
	AcademicYear int       `json:"academic_year"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Task Task `gorm:"foreignKey:TaskID" json:"-"`
}

// TaskSubmissionStatus represents a member's task submission status
type TaskSubmissionStatus struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	TaskID       int64      `json:"task_id"`
	MemberID     int64      `json:"member_id"`
	Status       string     `gorm:"default:'pending'" json:"status"` // pending, submitted, verified, graded
	SubmittedAt  *time.Time `json:"submitted_at,omitempty"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
	VerifiedByID *int64     `json:"verified_by_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relations
	Task   Task   `gorm:"foreignKey:TaskID" json:"-"`
	Member Member `gorm:"foreignKey:MemberID" json:"-"`
}

// Event represents a lab event or activity
type Event struct {
	ID            int64          `gorm:"primaryKey" json:"id"`
	Title         string         `gorm:"not null" json:"title"`
	Description   *string        `json:"description,omitempty"`
	EventType     string         `gorm:"not null" json:"event_type"` // workshop, seminar, social, competition
	Date          time.Time      `gorm:"type:date" json:"date"`
	StartTime     string         `gorm:"type:time;not null" json:"start_time"`
	EndTime       string         `gorm:"type:time;not null" json:"end_time"`
	Venue         *string        `json:"venue,omitempty"`
	ExternalLinks pq.StringArray `gorm:"type:text[]" json:"external_links"`
	CreatedByID   int64          `json:"created_by_id"`
	DeletedAt     *time.Time     `json:"deleted_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`

	// Relations
	CreatedBy User `gorm:"foreignKey:CreatedByID" json:"-"`
}

// FAQCategory represents a category for frequently asked questions
type FAQCategory struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Slug      string    `gorm:"not null;uniqueIndex" json:"slug"`
	Position  int       `gorm:"default:0" json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FAQ represents a frequently asked question
type FAQ struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	FAQCategoryID int64     `json:"faq_category_id"`
	Question      string    `gorm:"not null" json:"question"`
	Answer        string    `gorm:"type:text" json:"answer"`
	Position      int       `gorm:"default:0" json:"position"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Relations
	Category FAQCategory `gorm:"foreignKey:FAQCategoryID" json:"-"`
}

// ContactMessage represents a message from a member
type ContactMessage struct {
	ID                  int64     `gorm:"primaryKey" json:"id"`
	MemberID            int64     `json:"member_id"`
	Subject             string    `gorm:"not null" json:"subject"`
	Category            string    `gorm:"not null" json:"category"` // general, technical, administrative
	Message             string    `gorm:"type:text" json:"message"`
	Status              string    `gorm:"default:'new'" json:"status"` // new, in_progress, resolved
	EmailDeliveryStatus *string   `json:"email_delivery_status,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	// Relations
	Member Member `gorm:"foreignKey:MemberID" json:"-"`
}

// Poll represents a poll within a group message
type Poll struct {
	ID               int64      `gorm:"primaryKey" json:"id"`
	GroupMessageID   int64      `json:"group_message_id"`
	Question         string     `json:"question"`
	IsMultipleChoice bool       `json:"is_multiple_choice"`
	ClosesAt         *time.Time `json:"closes_at"`
	IsClosed         bool       `json:"is_closed"`
	CreatedByID      int64      `json:"created_by_id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	Options []PollOption `gorm:"foreignKey:PollID" json:"options"`
	Votes   []PollVote   `gorm:"foreignKey:PollID" json:"votes"`
}

// PollOption represents a poll option
type PollOption struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	PollID    int64     `json:"poll_id"`
	Text      string    `json:"text"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PollVote represents a poll vote
type PollVote struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	PollID       int64     `json:"poll_id"`
	PollOptionID int64     `json:"poll_option_id"`
	UserID       int64     `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GroupMessage represents a message in a group/channel
type GroupMessage struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	UserID          int64     `json:"user_id"`
	Content         *string   `json:"content,omitempty"`
	ReplyToID       *int64    `json:"reply_to_id,omitempty"`
	ForwardedFromID *int64    `json:"forwarded_from_id,omitempty"`
	IsEdited        bool      `gorm:"default:false" json:"is_edited"`
	IsDeleted       bool      `gorm:"default:false" json:"is_deleted"`
	IsPinned        bool      `gorm:"default:false" json:"is_pinned"`
	MessageType     string    `gorm:"default:'text'" json:"message_type"` // text, file, system
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relations
	User        User                `gorm:"foreignKey:UserID" json:"-"`
	Attachments []MessageAttachment `gorm:"foreignKey:GroupMessageID" json:"-"`
	Receipts    []MessageReceipt    `gorm:"foreignKey:GroupMessageID" json:"-"`
	Reactions   []MessageReaction   `gorm:"foreignKey:GroupMessageID" json:"-"`
	Poll        *Poll               `gorm:"foreignKey:GroupMessageID" json:"-"`
	LinkPreview *LinkPreview        `gorm:"foreignKey:GroupMessageID" json:"-"`
}

// MessageAttachment represents an attachment to a group message
type MessageAttachment struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	GroupMessageID int64     `json:"group_message_id"`
	Filename       string    `json:"filename"`
	ContentType    string    `json:"content_type"`
	ByteSize       int       `json:"byte_size"`
	StorageKey     string    `json:"storage_key"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MessageReceipt tracks message delivery/read status
type MessageReceipt struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	GroupMessageID int64      `json:"group_message_id"`
	UserID         int64      `json:"user_id"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// MessageReaction represents a reaction to a group message
type MessageReaction struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	GroupMessageID int64     `json:"group_message_id"`
	UserID         int64     `json:"user_id"`
	Emoji          string    `json:"emoji"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MessagePin represents a pinned message
type MessagePin struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	GroupMessageID int64     `json:"group_message_id"`
	PinnedByID     int64     `json:"pinned_by_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MessageStar represents a starred message
type MessageStar struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	GroupMessageID int64     `json:"group_message_id"`
	UserID         int64     `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// LinkPreview represents a link preview for shared URLs
type LinkPreview struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	GroupMessageID int64     `json:"group_message_id"`
	URL            string    `json:"url"`
	Title          *string   `json:"title,omitempty"`
	Description    *string   `json:"description,omitempty"`
	ImageURL       *string   `json:"image_url,omitempty"`
	SiteName       *string   `json:"site_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PlatformSetting represents platform-wide settings
type PlatformSetting struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"not null;uniqueIndex" json:"key"`
	Value     *string   `json:"value,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminTransfer represents an admin account transfer
type AdminTransfer struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	FromAdminID int64      `json:"from_admin_id"`
	ToAdminID   int64      `json:"to_admin_id"`
	Status      string     `gorm:"default:'pending'" json:"status"` // pending, completed, cancelled
	InitiatedAt time.Time  `json:"initiated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// LabRulesVersion represents a version of lab rules
type LabRulesVersion struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	VersionNumber int       `gorm:"not null;uniqueIndex" json:"version_number"`
	Content       string    `gorm:"type:text" json:"content"`
	IsCurrent     bool      `gorm:"default:false" json:"is_current"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// LabRulesAcceptance represents a member's acceptance of lab rules
type LabRulesAcceptance struct {
	ID                int64     `gorm:"primaryKey" json:"id"`
	MemberID          int64     `json:"member_id"`
	LabRulesVersionID int64     `json:"lab_rules_version_id"`
	AcceptedAt        time.Time `json:"accepted_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Relations
	Member          Member          `gorm:"foreignKey:MemberID" json:"-"`
	LabRulesVersion LabRulesVersion `gorm:"foreignKey:LabRulesVersionID" json:"-"`
}

// AuditLog represents an audit trail entry
type AuditLog struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	UserID       *int64         `json:"user_id,omitempty"`
	Action       string         `gorm:"not null" json:"action"`
	ResourceType string         `gorm:"not null" json:"resource_type"`
	ResourceID   *int64         `json:"resource_id,omitempty"`
	Details      datatypes.JSON `json:"details,omitempty"`
	IPAddress    *string        `json:"ip_address,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// GroupSetting stores discussion group configuration.
type GroupSetting struct {
	ID                  int64     `gorm:"primaryKey" json:"id"`
	Permission          string    `gorm:"not null;default:'everyone'" json:"permission"`
	Description         *string   `json:"description,omitempty"`
	MediaSharingEnabled bool      `gorm:"not null;default:true" json:"media_sharing_enabled"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// BackgroundJob is a durable queued job processed by the jobs worker.
type BackgroundJob struct {
	ID          int64          `gorm:"primaryKey" json:"id"`
	JobType     string         `gorm:"not null" json:"job_type"`
	Payload     datatypes.JSON `gorm:"type:jsonb;not null" json:"payload"`
	Status      string         `gorm:"not null;default:'pending'" json:"status"`
	Attempts    int            `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts int            `gorm:"not null;default:5" json:"max_attempts"`
	RunAt       time.Time      `gorm:"not null" json:"run_at"`
	LockedAt    *time.Time     `json:"locked_at,omitempty"`
	LastError   *string        `json:"last_error,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ContentBlock represents a content block for the website
type ContentBlock struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"not null;uniqueIndex" json:"key"`
	Title     *string   `json:"title,omitempty"`
	Content   *string   `json:"content,omitempty"`
	Position  int       `gorm:"default:0" json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
