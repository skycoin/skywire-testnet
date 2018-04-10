package db

import (
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"os"
)

var engine *xorm.Engine
var dbName = "skywire-discovery.db"

func Init() (err error) {
	if _, err = os.Stat(dbName); err == nil {
		err = os.Remove(dbName)
		if err != nil {
			return
		}
	}
	engine, err = xorm.NewEngine("sqlite3", dbName)
	if err != nil {
		return
	}
	engine.SetMaxIdleConns(30)
	engine.SetMaxOpenConns(30)
	engine.ShowSQL(true)
	err = engine.Ping()
	if err != nil {
		return
	}
	err = createTables()
	return
}

func createTables() (err error) {
	nodeTable := `CREATE TABLE node (
	id              INTEGER    PRIMARY KEY AUTOINCREMENT
	NOT NULL,
		[key]           CHAR (66),
		service_address CHAR (50),
		location        CHAR (100),
		version         TEXT,
		priority		INTEGER,
		created         DATETIME,
		updated         DATETIME
	);`
	serviceTable := `CREATE TABLE service (
    id                  INTEGER   PRIMARY KEY AUTOINCREMENT
                                  NOT NULL,
    [key]               CHAR (66),
    address             CHAR (50),
    hide_from_discovery INTEGER,
    allow_nodes         TEXT,
    version             CHAR (10),
	created         DATETIME,
	updated         DATETIME,
    node_id             INTEGER,
    FOREIGN KEY (
        node_id
    )
    REFERENCES node (id) ON DELETE CASCADE
	);`
	attributesTable := `CREATE TABLE attributes (
    name       CHAR (20),
    service_id INTEGER,
    FOREIGN KEY (
        service_id
    )
    REFERENCES service (id) ON DELETE CASCADE
	);`

	exist, err := engine.IsTableExist("node")
	if err != nil {
		log.Errorf("check table exist err: %s", err)
		return
	}
	if !exist {
		_, err := engine.Exec(nodeTable)
		if err != nil {
			log.Errorf("create node table err: %s", err)
			return err
		}
		nodeIndex := `CREATE UNIQUE INDEX IDX_node_key ON node (
		"key"
		);`
		_, err = engine.Exec(nodeIndex)
		if err != nil {
			log.Errorf("create node index err: %s", err)
			return err
		}
	}
	exist, err = engine.IsTableExist("service")
	if err != nil {
		log.Errorf("check table exist err: %s", err)
		return
	}
	if !exist {
		_, err := engine.Exec(serviceTable)
		if err != nil {
			log.Errorf("create service table err: %s", err)
			return err
		}
		serviceIndex := `CREATE UNIQUE INDEX IDX_service_key ON service (
		"key"
		);`
		serviceNodeIdIndex := `CREATE INDEX IDX_service_node_id ON service (
			node_id
		);`
		_, err = engine.Exec(serviceIndex)
		if err != nil {
			log.Errorf("create service key index err: %s", err)
			return err
		}
		_, err = engine.Exec(serviceNodeIdIndex)
		if err != nil {
			log.Errorf("create service node id index err: %s", err)
			return err
		}
	}
	exist, err = engine.IsTableExist("attributes")
	if err != nil {
		log.Errorf("check table exist err: %s", err)
		return
	}
	if !exist {
		_, err := engine.Exec(attributesTable)
		if err != nil {
			log.Errorf("create attributes table err: %s", err)
			return err
		}
		attributesNameIndex := `CREATE INDEX IDX_attributes_name ON attributes (
			name
		);`
		attributesServiceIdIndex := `CREATE INDEX IDX_attributes_service_id ON attributes (
			service_id
		);`
		_, err = engine.Exec(attributesNameIndex)
		if err != nil {
			log.Errorf("create attributes Name Index err: %s", err)
			return err
		}
		_, err = engine.Exec(attributesServiceIdIndex)
		if err != nil {
			log.Errorf("create attributes ServiceId Index err: %s", err)
			return err
		}
	}
	return
}
