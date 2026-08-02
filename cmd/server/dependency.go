package main

import "github.com/liebeSonne/gophkeeper/internal/repository/db"

type dependencyContainer struct {
}

func newDependencyContainer(con *connectionContainer) *dependencyContainer {
	pool := con.Database.Pool()

	userRepo := db.NewUserRepo(pool)
	tokenRepo := db.NewTokenRepo(pool)
	dataRepo := db.NewDataRepo(pool)

	_ = userRepo
	_ = tokenRepo
	_ = dataRepo

	return &dependencyContainer{}
}
