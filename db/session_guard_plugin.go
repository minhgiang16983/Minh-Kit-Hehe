package db

import (
	"reflect"

	"gorm.io/gorm"
)

const sessionGuardBeforeQuery = "hsk:session_guard:before_query"
const sessionGuardAfter = "hsk:session_guard:after"

// sessionGuardPlugin prevents leaked GORM session flags (e.g. RaiseErrorOnNotFound)
// on a shared *gorm.DB from affecting unrelated queries in other goroutines/handlers.
//
// Background: mutating db.Statement on the root *gorm.DB (e.g. tx.Statement.RaiseErrorOnNotFound = true
// when tx is GetMariaDb()) copies into later WithContext()/Find() chains and turns empty Find results
// into gorm.ErrRecordNotFound.
type sessionGuardPlugin struct {
	root *gorm.DB
}

func newSessionGuardPlugin() *sessionGuardPlugin {
	return &sessionGuardPlugin{}
}

func (p *sessionGuardPlugin) Name() string {
	return "hsk-session-guard"
}

func (p *sessionGuardPlugin) Initialize(db *gorm.DB) error {
	p.root = db

	if err := db.Callback().Query().Before("gorm:query").Register(sessionGuardBeforeQuery, guardBeforeQuery); err != nil {
		return err
	}

	after := func(tx *gorm.DB) { p.resetRootSessionFlags() }
	for _, register := range []func(string, func(*gorm.DB)) error{
		func(name string, fn func(*gorm.DB)) error { return db.Callback().Query().After("gorm:query").Register(name, fn) },
		func(name string, fn func(*gorm.DB)) error { return db.Callback().Create().After("gorm:create").Register(name, fn) },
		func(name string, fn func(*gorm.DB)) error { return db.Callback().Update().After("gorm:update").Register(name, fn) },
		func(name string, fn func(*gorm.DB)) error { return db.Callback().Delete().After("gorm:delete").Register(name, fn) },
		func(name string, fn func(*gorm.DB)) error { return db.Callback().Row().After("gorm:row").Register(name, fn) },
		func(name string, fn func(*gorm.DB)) error { return db.Callback().Raw().After("gorm:raw").Register(name, fn) },
	} {
		if err := register(sessionGuardAfter, after); err != nil {
			return err
		}
	}

	return nil
}

func (p *sessionGuardPlugin) resetRootSessionFlags() {
	if p.root == nil || p.root.Statement == nil {
		return
	}
	p.root.Statement.RaiseErrorOnNotFound = false
}

// ResetSessionFlags clears leaked GORM statement flags on a shared *gorm.DB.
// Prefer relying on sessionGuardPlugin registered by db.New; call this only when
// code mutates db.Statement directly without running a query afterward.
func ResetSessionFlags(db *gorm.DB) {
	if db == nil || db.Statement == nil {
		return
	}
	db.Statement.RaiseErrorOnNotFound = false
}

func guardBeforeQuery(tx *gorm.DB) {
	stmt := tx.Statement
	if stmt == nil {
		return
	}

	// Collection reads (Find on slice) must never surface ErrRecordNotFound for empty results.
	if destIsSlice(stmt.Dest) {
		stmt.RaiseErrorOnNotFound = false
		return
	}

	// First/Take/Last set RaiseErrorOnNotFound before the query callback runs.
	if stmt.RaiseErrorOnNotFound {
		return
	}

	// Find on a single struct/map without LIMIT 1 should behave like a plain Find.
	stmt.RaiseErrorOnNotFound = false
}

func destIsSlice(dest interface{}) bool {
	if dest == nil {
		return false
	}
	v := reflect.ValueOf(dest)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}
