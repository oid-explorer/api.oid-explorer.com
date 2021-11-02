package database

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/oid-explorer/api.oid-explorer.com/oid"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"strconv"
	"sync"
)

var db struct {
	sync.Once
	Database
}

type Database interface {
	GetOID(string) (oid.OID, error)
	GetOIDRelation(string) (oid.Relation, error)
	GetOIDParent(string) (oid.OID, error)
	GetOIDSiblings(string) ([]oid.OID, error)
	GetOIDChildren(string) ([]oid.OID, error)
	SearchOID(OIDSearch) ([]oid.OID, error)
}

type database struct {
	*sqlx.DB
}

func initDB() error {
	d, err := sqlx.Connect("mysql", viper.GetString("datasourcename"))
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	db.Database = database{
		DB: d,
	}
	return nil
}

func GetDB() (Database, error) {
	var err error
	db.Do(func() {
		err = initDB()
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize DB")
	}
	if db.Database == nil {
		return nil, errors.New("database was not initialized")
	}
	return db.Database, nil
}

func (d database) GetOID(wantedOID string) (oid.OID, error) {
	if wantedOID == "" {
		return oid.OID{}, errors.New("got empty oid to search for")
	}

	var res oid.OID
	err := d.Get(&res, "SELECT name, oid, objectType "+
		"FROM oids "+
		"WHERE oid = ?", wantedOID)
	if err != nil {
		return oid.OID{}, errors.Wrap(err, "failed to query database for oid")
	}

	var descriptions []oid.Description
	err = d.Select(&descriptions, "SELECT m.name AS mib, d.description AS description "+
		"FROM oidDescriptions d "+
		"JOIN oids o ON d.oid = o.id "+
		"JOIN mibs m ON d.mib = m.id "+
		"WHERE o.oid = ?", wantedOID)
	if err != nil {
		return oid.OID{}, errors.Wrap(err, "failed to query database for description")
	}
	res.Descriptions = descriptions

	parent, err := d.GetOIDParent(wantedOID)
	if err == nil {
		res.Parent = &parent
	}

	return res, nil
}

func (d database) GetOIDRelation(wantedOID string) (oid.Relation, error) {
	if wantedOID == "" {
		return oid.Relation{}, errors.New("got empty oid to search for")
	}

	currentOID, err := d.GetOID(wantedOID)
	if err != nil {
		return oid.Relation{}, err
	}

	var childrenRelation []oid.Relation
	children, err := d.GetOIDChildren(wantedOID)
	if err != nil {
		return oid.Relation{}, err
	}
	for _, child := range children {
		childrenRelation = append(childrenRelation, oid.Relation{
			OID: oid.OID{
				OID:  child.OID,
				Name: child.Name,
			},
		},
		)
	}

	res := oid.Relation{
		OID: oid.OID{
			OID:  currentOID.OID,
			Name: currentOID.Name,
		},
		Children: childrenRelation,
	}

	for {
		if currentOID.Parent == nil {
			break
		}

		res = oid.Relation{
			OID: oid.OID{
				Name: currentOID.Parent.Name,
				OID:  currentOID.Parent.OID,
			},
			Children: []oid.Relation{res},
		}

		currentOID, err = d.GetOID(currentOID.Parent.OID)
		if err != nil {
			return oid.Relation{}, err
		}
	}

	return res, nil
}

func (d database) GetOIDParent(wantedOID string) (oid.OID, error) {
	if wantedOID == "" {
		return oid.OID{}, errors.New("got empty oid to search for")
	}

	var parent oid.OID
	err := d.Get(&parent, "SELECT p.name AS name, p.oid AS oid "+
		"FROM oids o "+
		"LEFT JOIN oids p ON o.parent = p.id "+
		"WHERE o.oid = ?", wantedOID)
	if err != nil {
		return oid.OID{}, errors.Wrap(err, "failed to query database for parent")
	}

	return parent, nil
}

func (d database) GetOIDSiblings(wantedOID string) ([]oid.OID, error) {
	if wantedOID == "" {
		return nil, errors.New("got empty oid to search for")
	}

	var res []oid.OID
	err := d.Select(&res, "SELECT name, oid "+
		"FROM oids "+
		"WHERE parent = (SELECT parent FROM oids WHERE oid = ?) AND oid != ? "+
		"ORDER BY LENGTH(oid), oid", wantedOID, wantedOID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query database")
	}

	return res, nil
}

type OIDSearch struct {
	Any   *string
	Name  *string
	OID   *string
	Limit *int
}

func (d database) GetOIDChildren(wantedOID string) ([]oid.OID, error) {
	if wantedOID == "" {
		return nil, errors.New("got empty oid to search for")
	}

	var res []oid.OID
	err := d.Select(&res, "SELECT o.name, o.oid "+
		"FROM oids o "+
		"LEFT JOIN oids p ON o.parent = p.id "+
		"WHERE p.oid = ? "+
		"ORDER BY LENGTH(o.oid), o.oid", wantedOID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query database")
	}

	return res, nil
}

func (d database) SearchOID(search OIDSearch) ([]oid.OID, error) {
	selectQuery := "SELECT name, oid FROM oids "
	var limitQuery string
	if search.Limit != nil {
		limitQuery = " LIMIT " + strconv.Itoa(*search.Limit)
	}

	var res []oid.OID
	var err error
	if search.Any != nil {
		err = d.Select(&res, selectQuery+
			"WHERE name LIKE ? OR oid LIKE ? ORDER BY INSTR(name, ?), INSTR(oid, ?), name"+
			limitQuery, "%"+*search.Any+"%", "%"+*search.Any+"%", *search.Any, *search.Any)
	} else if search.Name != nil {
		err = d.Select(&res, selectQuery+
			"WHERE name LIKE ? ORDER BY INSTR(name, ?), name"+
			limitQuery, "%"+*search.Name+"%", *search.Name)
	} else if search.OID != nil {
		err = d.Select(&res, selectQuery+
			"WHERE oid LIKE ? ORDER BY INSTR(oid, ?), name"+
			limitQuery, "%"+*search.OID+"%", *search.OID)
	} else {
		err = d.Select(&res, selectQuery+
			"ORDER BY oid"+
			limitQuery)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to query database")
	}

	return res, nil
}
