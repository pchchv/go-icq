package state

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
)

//go:embed migrations/*
var migrations embed.FS

// SQLiteUserStore stores user feedbag (buddy list), profile,
// and authentication credentials information in a SQLite database.
type SQLiteUserStore struct {
	db *sql.DB
}

// NewSQLiteUserStore creates a new instance of SQLiteUserStore.
// If the database does not already exist,
// a new one is created with the required schema.
func NewSQLiteUserStore(dbFilePath string) (*SQLiteUserStore, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys=on", dbFilePath))
	if err != nil {
		return nil, err
	}

	// Set the maximum number of open connections to 1.
	// This is crucial to prevent SQLITE_BUSY errors,
	// which occur when the database is locked due to concurrent access.
	// By limiting the number of open connections to 1,
	// we ensure that all database operations are serialized,
	// thus avoiding any potential locking issues.
	db.SetMaxOpenConns(1)

	store := &SQLiteUserStore{db: db}
	if err := store.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

func (f SQLiteUserStore) User(ctx context.Context, screenName IdentScreenName) (*User, error) {
	users, err := f.queryUsers(ctx, `identScreenName = ?`, []any{screenName.String()})
	if err != nil {
		return nil, fmt.Errorf("User: %w", err)
	}

	if len(users) == 0 {
		return nil, nil
	}

	return &users[0], nil
}

func (f SQLiteUserStore) AllUsers(ctx context.Context) ([]User, error) {
	q := `SELECT identScreenName, displayScreenName, isICQ, isBot FROM users`
	rows, err := f.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var identSN, displaySN string
		var isICQ, isBot bool
		if err := rows.Scan(&identSN, &displaySN, &isICQ, &isBot); err != nil {
			return nil, err
		}
		users = append(users, User{
			IdentScreenName:   NewIdentScreenName(identSN),
			DisplayScreenName: DisplayScreenName(displaySN),
			IsICQ:             isICQ,
			IsBot:             isBot,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (f SQLiteUserStore) FindByUIN(ctx context.Context, UIN uint32) (User, error) {
	users, err := f.queryUsers(ctx, `identScreenName = ?`, []any{strconv.Itoa(int(UIN))})
	if err != nil {
		return User{}, fmt.Errorf("FindByUIN: %w", err)
	}

	if len(users) == 0 {
		return User{}, ErrNoUser
	}

	return users[0], nil
}

func (f SQLiteUserStore) FindByICQEmail(ctx context.Context, email string) (User, error) {
	users, err := f.queryUsers(ctx, `icq_basicInfo_emailAddress = ?`, []any{email})
	if err != nil {
		return User{}, fmt.Errorf("FindByICQEmail: %w", err)
	}

	if len(users) == 0 {
		return User{}, ErrNoUser
	}

	return users[0], nil
}

func (f SQLiteUserStore) FindByICQName(ctx context.Context, firstName, lastName, nickName string) ([]User, error) {
	var args []any
	var clauses []string
	if firstName != "" {
		args = append(args, firstName)
		clauses = append(clauses, `LOWER(icq_basicInfo_firstName) = LOWER(?)`)
	}

	if lastName != "" {
		args = append(args, lastName)
		clauses = append(clauses, `LOWER(icq_basicInfo_lastName) = LOWER(?)`)
	}

	if nickName != "" {
		args = append(args, nickName)
		clauses = append(clauses, `LOWER(icq_basicInfo_nickName) = LOWER(?)`)
	}

	whereClause := strings.Join(clauses, " AND ")
	users, err := f.queryUsers(ctx, whereClause, args)
	if err != nil {
		return users, fmt.Errorf("FindByICQName: %w", err)
	}

	return users, nil
}

func (f SQLiteUserStore) FindByICQInterests(ctx context.Context, code uint16, keywords []string) ([]User, error) {
	var args []any
	var clauses []string
	for i := 1; i <= 4; i++ {
		var subClauses []string
		args = append(args, code)
		for _, key := range keywords {
			subClauses = append(subClauses, fmt.Sprintf("icq_interests_keyword%d LIKE ?", i))
			args = append(args, "%"+key+"%")
		}
		clauses = append(clauses, fmt.Sprintf("(icq_interests_code%d = ? AND (%s))", i, strings.Join(subClauses, " OR ")))
	}

	cond := strings.Join(clauses, " OR ")
	users, err := f.queryUsers(ctx, cond, args)
	if err != nil {
		return users, fmt.Errorf("FindByICQInterests: %w", err)
	}

	return users, nil
}

func (f SQLiteUserStore) FindByICQKeyword(ctx context.Context, keyword string) ([]User, error) {
	var args []any
	var clauses []string
	for i := 1; i <= 4; i++ {
		args = append(args, "%"+keyword+"%")
		clauses = append(clauses, fmt.Sprintf("icq_interests_keyword%d LIKE ?", i))
	}

	whereClause := strings.Join(clauses, " OR ")
	users, err := f.queryUsers(ctx, whereClause, args)
	if err != nil {
		return users, fmt.Errorf("FindByICQKeyword: %w", err)
	}

	return users, nil
}

func (f SQLiteUserStore) FindByAIMNameAndAddr(ctx context.Context, info AIMNameAndAddr) ([]User, error) {
	var args []any
	var clauses []string
	if info.FirstName != "" {
		args = append(args, info.FirstName)
		clauses = append(clauses, `LOWER(aim_firstName) = LOWER(?)`)
	}

	if info.LastName != "" {
		args = append(args, info.LastName)
		clauses = append(clauses, `LOWER(aim_lastName) = LOWER(?)`)
	}

	if info.MiddleName != "" {
		args = append(args, info.MiddleName)
		clauses = append(clauses, `LOWER(aim_middleName) = LOWER(?)`)
	}

	if info.MaidenName != "" {
		args = append(args, info.MaidenName)
		clauses = append(clauses, `LOWER(aim_maidenName) = LOWER(?)`)
	}

	if info.Country != "" {
		args = append(args, info.Country)
		clauses = append(clauses, `LOWER(aim_country) = LOWER(?)`)
	}

	if info.State != "" {
		args = append(args, info.State)
		clauses = append(clauses, `LOWER(aim_state) = LOWER(?)`)
	}

	if info.City != "" {
		args = append(args, info.City)
		clauses = append(clauses, `LOWER(aim_city) = LOWER(?)`)
	}

	if info.NickName != "" {
		args = append(args, info.NickName)
		clauses = append(clauses, `LOWER(aim_nickName) = LOWER(?)`)
	}

	if info.ZIPCode != "" {
		args = append(args, info.ZIPCode)
		clauses = append(clauses, `LOWER(aim_zipCode) = LOWER(?)`)
	}

	if info.Address != "" {
		args = append(args, info.Address)
		clauses = append(clauses, `LOWER(aim_address) = LOWER(?)`)
	}

	whereClause := strings.Join(clauses, " AND ")
	users, err := f.queryUsers(ctx, whereClause, args)
	if err != nil {
		return users, fmt.Errorf("FindByAIMNameAndAddr: %w", err)
	}

	return users, nil
}

func (f SQLiteUserStore) FindByAIMEmail(ctx context.Context, email string) (User, error) {
	users, err := f.queryUsers(ctx, `emailAddress = ?`, []any{email})
	if err != nil {
		return User{}, fmt.Errorf("FindByAIMEmail: %w", err)
	}

	if len(users) == 0 {
		return User{}, ErrNoUser
	}

	return users[0], nil
}

func (f SQLiteUserStore) FindByAIMKeyword(ctx context.Context, keyword string) ([]User, error) {
	where := `
		(SELECT id FROM aimKeyword WHERE name = ?) IN
		(aim_keyword1, aim_keyword2, aim_keyword3, aim_keyword4, aim_keyword5)
	`
	users, err := f.queryUsers(ctx, where, []any{keyword})
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (us SQLiteUserStore) runMigrations() error {
	migrationFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to prepare migration subdirectory: %v", err)
	}

	sourceInstance, err := httpfs.New(http.FS(migrationFS), ".")
	if err != nil {
		return fmt.Errorf("failed to create source instance from embedded filesystem: %v", err)
	}

	driver, err := migratesqlite.WithInstance(us.db, &migratesqlite.Config{})
	if err != nil {
		return fmt.Errorf("cannot create database driver: %v", err)
	}

	m, err := migrate.NewWithInstance("httpfs", sourceInstance, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	return nil
}

// queryUsers retrieves a list of users from the database based on the
// specified WHERE clause and query parameters.
// Returns a slice of User objects or an error if the query fails.
func (us SQLiteUserStore) queryUsers(ctx context.Context, whereClause string, queryParams []any) ([]User, error) {
	q := `
		SELECT
			identScreenName,
			displayScreenName,
			emailAddress,
			authKey,
			strongMD5Pass,
			weakMD5Pass,
			confirmStatus,
			regStatus,
			suspendedStatus,
			isBot,
			isICQ,
			icq_affiliations_currentCode1,
			icq_affiliations_currentCode2,
			icq_affiliations_currentCode3,
			icq_affiliations_currentKeyword1,
			icq_affiliations_currentKeyword2,
			icq_affiliations_currentKeyword3,
			icq_affiliations_pastCode1,
			icq_affiliations_pastCode2,
			icq_affiliations_pastCode3,
			icq_affiliations_pastKeyword1,
			icq_affiliations_pastKeyword2,
			icq_affiliations_pastKeyword3,
			icq_basicInfo_address,
			icq_basicInfo_cellPhone,
			icq_basicInfo_city,
			icq_basicInfo_countryCode,
			icq_basicInfo_emailAddress,
			icq_basicInfo_fax,
			icq_basicInfo_firstName,
			icq_basicInfo_gmtOffset,
			icq_basicInfo_lastName,
			icq_basicInfo_nickName,
			icq_basicInfo_phone,
			icq_basicInfo_publishEmail,
			icq_basicInfo_state,
			icq_basicInfo_zipCode,
			icq_interests_code1,
			icq_interests_code2,
			icq_interests_code3,
			icq_interests_code4,
			icq_interests_keyword1,
			icq_interests_keyword2,
			icq_interests_keyword3,
			icq_interests_keyword4,
			icq_moreInfo_birthDay,
			icq_moreInfo_birthMonth,
			icq_moreInfo_birthYear,
			icq_moreInfo_gender,
			icq_moreInfo_homePageAddr,
			icq_moreInfo_lang1,
			icq_moreInfo_lang2,
			icq_moreInfo_lang3,
			icq_notes,
			icq_permissions_authRequired,
			icq_workInfo_address,
			icq_workInfo_city,
			icq_workInfo_company,
			icq_workInfo_countryCode,
			icq_workInfo_department,
			icq_workInfo_fax,
			icq_workInfo_occupationCode,
			icq_workInfo_phone,
			icq_workInfo_position,
			icq_workInfo_state,
			icq_workInfo_webPage,
			icq_workInfo_zipCode,
			aim_firstName,
			aim_lastName,
			aim_middleName,
			aim_maidenName,
			aim_country,
			aim_state,
			aim_city,
			aim_nickName,
			aim_zipCode,
			aim_address,
			tocConfig,
			lastWarnUpdate,
			lastWarnLevel,
			offlineMsgCount
		FROM users
		WHERE %s
	`
	q = fmt.Sprintf(q, whereClause)
	rows, err := us.db.QueryContext(ctx, q, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var sn string
		var lastWarnUpdateUnix int64
		err := rows.Scan(
			&sn,
			&u.DisplayScreenName,
			&u.EmailAddress,
			&u.AuthKey,
			&u.StrongMD5Pass,
			&u.WeakMD5Pass,
			&u.ConfirmStatus,
			&u.RegStatus,
			&u.SuspendedStatus,
			&u.IsBot,
			&u.IsICQ,
			&u.ICQAffiliations.CurrentCode1,
			&u.ICQAffiliations.CurrentCode2,
			&u.ICQAffiliations.CurrentCode3,
			&u.ICQAffiliations.CurrentKeyword1,
			&u.ICQAffiliations.CurrentKeyword2,
			&u.ICQAffiliations.CurrentKeyword3,
			&u.ICQAffiliations.PastCode1,
			&u.ICQAffiliations.PastCode2,
			&u.ICQAffiliations.PastCode3,
			&u.ICQAffiliations.PastKeyword1,
			&u.ICQAffiliations.PastKeyword2,
			&u.ICQAffiliations.PastKeyword3,
			&u.ICQBasicInfo.Address,
			&u.ICQBasicInfo.CellPhone,
			&u.ICQBasicInfo.City,
			&u.ICQBasicInfo.CountryCode,
			&u.ICQBasicInfo.EmailAddress,
			&u.ICQBasicInfo.Fax,
			&u.ICQBasicInfo.FirstName,
			&u.ICQBasicInfo.GMTOffset,
			&u.ICQBasicInfo.LastName,
			&u.ICQBasicInfo.Nickname,
			&u.ICQBasicInfo.Phone,
			&u.ICQBasicInfo.PublishEmail,
			&u.ICQBasicInfo.State,
			&u.ICQBasicInfo.ZIPCode,
			&u.ICQInterests.Code1,
			&u.ICQInterests.Code2,
			&u.ICQInterests.Code3,
			&u.ICQInterests.Code4,
			&u.ICQInterests.Keyword1,
			&u.ICQInterests.Keyword2,
			&u.ICQInterests.Keyword3,
			&u.ICQInterests.Keyword4,
			&u.ICQMoreInfo.BirthDay,
			&u.ICQMoreInfo.BirthMonth,
			&u.ICQMoreInfo.BirthYear,
			&u.ICQMoreInfo.Gender,
			&u.ICQMoreInfo.HomePageAddr,
			&u.ICQMoreInfo.Lang1,
			&u.ICQMoreInfo.Lang2,
			&u.ICQMoreInfo.Lang3,
			&u.ICQNotes.Notes,
			&u.ICQPermissions.AuthRequired,
			&u.ICQWorkInfo.Address,
			&u.ICQWorkInfo.City,
			&u.ICQWorkInfo.Company,
			&u.ICQWorkInfo.CountryCode,
			&u.ICQWorkInfo.Department,
			&u.ICQWorkInfo.Fax,
			&u.ICQWorkInfo.OccupationCode,
			&u.ICQWorkInfo.Phone,
			&u.ICQWorkInfo.Position,
			&u.ICQWorkInfo.State,
			&u.ICQWorkInfo.WebPage,
			&u.ICQWorkInfo.ZIPCode,
			&u.AIMDirectoryInfo.FirstName,
			&u.AIMDirectoryInfo.LastName,
			&u.AIMDirectoryInfo.MiddleName,
			&u.AIMDirectoryInfo.MaidenName,
			&u.AIMDirectoryInfo.Country,
			&u.AIMDirectoryInfo.State,
			&u.AIMDirectoryInfo.City,
			&u.AIMDirectoryInfo.NickName,
			&u.AIMDirectoryInfo.ZIPCode,
			&u.AIMDirectoryInfo.Address,
			&u.TOCConfig,
			&lastWarnUpdateUnix,
			&u.LastWarnLevel,
			&u.OfflineMsgCount,
		)
		if err != nil {
			return nil, err
		}

		u.IdentScreenName = NewIdentScreenName(sn)
		u.LastWarnUpdate = time.Unix(lastWarnUpdateUnix, 0).UTC()
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
