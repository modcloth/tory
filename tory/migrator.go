package tory

var (
	databaseMigrations = map[string][]string{
		"2014-08-01T19:18:12": []string{
			`CREATE EXTENSION IF NOT EXISTS hstore`,
			`CREATE SEQUENCE hosts_serial`,
			`CREATE TABLE IF NOT EXISTS hosts (
				id integer PRIMARY KEY DEFAULT nextval('hosts_serial'),
				name varchar(255) UNIQUE NOT NULL,
				package varchar(255),
				image varchar(255),
				type varchar(255),
				ip inet NOT NULL,
				tags hstore,
				vars hstore,
				modified timestamp DEFAULT current_timestamp
			)`,
			`CREATE INDEX hosts_package_idx ON hosts (package)`,
			`CREATE INDEX hosts_image_idx ON hosts (image)`,
			`CREATE INDEX hosts_type_idx ON hosts (type)`,
			`CREATE INDEX hosts_ip_idx ON hosts (ip)`,
			`CREATE INDEX hosts_tags_idx ON hosts USING GIN (tags)`,
			`CREATE INDEX hosts_vars_idx ON hosts USING GIN (vars)`,
		},
	}
)

func MigrateMain(dbConnStr string) {
	db, err := newDatabase(dbConnStr)
	if err != nil {
		toryLog.Fatal(err.Error())
	}

	err = db.Setup(databaseMigrations)
	if err != nil {
		toryLog.Fatal(err.Error())

	}

	toryLog.Info("ding")
}
