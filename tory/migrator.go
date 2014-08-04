package tory

func MigrateMain(dbConnStr string) {
	db, err := newDatabase(dbConnStr, nil)
	if err != nil {
		toryLog.Fatal(err.Error())
	}

	err = db.Setup()
	if err != nil {
		toryLog.Fatal(err.Error())

	}

	toryLog.Info("ding")
}
