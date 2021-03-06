package repo

import (
	"fmt"
	"time"
	"database/sql"

	"github.com/aewens/nautical/cargo/model"
)

type Internal struct {
	Store  *sql.DB
	Crates map[int64]*model.Internal
}

func NewInternal(store *sql.DB) *Internal {
	return &Internal{
		Store:  store,
		Crates: make(map[int64]*model.Internal),
	}
}

func (self *Internal) Create() (model.Entity, error) {
	return model.NewInternal(self.Store)
}

func (self *Internal) Load(stream Stream) {
	for entity := range stream {
		internal, ok := entity.(*model.Internal)
		if !ok {
			continue
		}

		self.Crates[internal.ID] = internal
	}
}

func (self *Internal) Import(
	id      int64,
	uuid    []byte,
	added   time.Time,
	updated time.Time,
	flag    uint8,
	itype   string,
	origin  string,
	data    []byte,
) (model.Entity, error) {
	entity, err := self.Create()
	if err != nil {
		return entity, err
	}

	internal, ok := entity.(*model.Internal)
	if !ok {
		return entity, fmt.Errorf("Cannot cast to Internal: %#v", entity)
	}

	internal.ID = id
	internal.UUID = uuid
	internal.Added = added
	internal.Updated = updated
	internal.Flag = flag
	internal.Type = itype
	internal.Origin = origin
	internal.Data = data

	return entity, nil
}

func (self *Internal) Get(id int64) (model.Entity, error) {
	statement, err := self.Store.Prepare(`
		SELECT uuid, added, updated, flag, type, origin, data
		FROM internal WHERE id = ?;
	`)

	if err != nil {
		return nil, err
	}

	var (
		uuid    []byte
		added   time.Time
		updated time.Time
		flag    uint8
		itype   string
		origin  string
		data    []byte
	)

	defer statement.Close()
	err = statement.QueryRow(id).Scan(
		&uuid,
		&added,
		&updated,
		&flag,
		&itype,
		&origin,
		&data,
	)

	if err != nil {
		return nil, err
	}

	return self.Import(
		id,
		uuid,
		added,
		updated,
		flag,
		itype,
		origin,
		data,
	)
}

func (self *Internal) Process(stream Stream, rows *sql.Rows) {
	defer rows.Close()
	for rows.Next() {
		var (
			id      int64
			uuid    []byte
			added   time.Time
			updated time.Time
			flag    uint8
			itype   string
			origin  string
			data    []byte
		)

		err := rows.Scan(
			&id,
			&uuid,
			&added,
			&updated,
			&flag,
			&itype,
			&origin,
			&data,
		)

		if err != nil {
			continue
		}

		entity, err := self.Import(
			id,
			uuid,
			added,
			updated,
			flag,
			itype,
			origin,
			data,
		)

		if err != nil {
			continue
		}

		stream <- entity
	}

	close(stream)
}

func (self *Internal) All() Stream {
	stream := make(Stream)

	go func() {
		rows, err := self.Store.Query(`
			SELECT id, uuid, added, updated, flag, type, origin, data
			FROM internal;
		`)

		if err != nil {
			return
		}

		self.Process(stream, rows)
	}()

	return stream
}

func (self *Internal) Lookup(ids ...int64) Stream {
	stream := make(Stream)

	go func() {
		for _, id := range ids {
			entity, err := self.Get(id)
			if err == nil {
				stream <- entity
			}
		}

		close(stream)
	}()

	return stream
}

func (self *Internal) Contains(field string, search string) Stream {
	stream := make(Stream)

	go func() {
		statement, err := self.Store.Prepare(fmt.Sprintf(`
			SELECT id, uuid, added, updated, flag, type, origin, data
			FROM internal WHERE %s LIKE ?;
		`, field))

		if err != nil {
			return
		}

		defer statement.Close()
		rows, err := statement.Query("%" + search + "%")

		if err != nil {
			return
		}

		self.Process(stream, rows)
	}()

	return stream
}

func (self *Internal) Equals(field string, search string) Stream {
	stream := make(Stream)

	go func() {
		statement, err := self.Store.Prepare(fmt.Sprintf(`
			SELECT id, uuid, added, updated, flag, type, origin, data
			FROM internal WHERE %s = ?;
		`, field))

		if err != nil {
			return
		}

		defer statement.Close()
		rows, err := statement.Query(search)

		if err != nil {
			return
		}

		self.Process(stream, rows)
	}()

	return stream
}

func (self *Internal) Before(field string, search time.Time) Stream {
	stream := make(Stream)

	go func() {
		statement, err := self.Store.Prepare(fmt.Sprintf(`
			SELECT id, uuid, added, updated, flag, type, origin, data
			FROM internal WHERE %s < ?;
		`, field))

		if err != nil {
			return
		}

		defer statement.Close()
		rows, err := statement.Query(search)

		if err != nil {
			return
		}

		self.Process(stream, rows)
	}()

	return stream
}

func (self *Internal) After(field string, search time.Time) Stream {
	stream := make(Stream)

	go func() {
		statement, err := self.Store.Prepare(fmt.Sprintf(`
			SELECT id, uuid, added, updated, flag, type, origin, data
			FROM internal WHERE %s > ?;
		`, field))

		if err != nil {
			return
		}

		defer statement.Close()
		rows, err := statement.Query(search)

		if err != nil {
			return
		}

		self.Process(stream, rows)
	}()

	return stream
}

func (self *Internal) Between(
	field string,
	before time.Time,
	after time.Time,
) Stream {
	stream := make(Stream)

	go func() {
		statement, err := self.Store.Prepare(fmt.Sprintf(`
			SELECT id, uuid, added, updated, flag, type, origin, data
			FROM internal WHERE %s > ? AND %s < ?;
		`, field, field))

		if err != nil {
			return
		}

		defer statement.Close()
		rows, err := statement.Query(before, after)

		if err != nil {
			return
		}

		self.Process(stream, rows)
	}()

	return stream
}
