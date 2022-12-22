package controllers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-logr/logr"
	_ "github.com/lib/pq"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	sdev1beta1 "sde.domain/sdeController/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type sslMode bool

var ctxlog logr.Logger

func (s sslMode) String() string {
	switch s {
	case true:
		return "enabled"
	case false:
		return "disabled"
	}
	return "unknown"
}

type PGConnector struct {
	Host     string
	Port     string
	User     string
	Password string
	Dbname   string
	Sslmode  sslMode
}

func (p *PGConnector) Connect() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Dbname, p.Sslmode.String())
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		ctxlog.Info("Failed to connect to DB")
		return nil, err
	}

	return db, nil
}

func (r *SdeReconciler) PGCleanup(ctx context.Context, sde *sdev1beta1.Sde) error {
	ctxlog = log.FromContext(ctx)
	ctxlog.Info("Running pg cleanup")

	// Get connection strings
	dbSecret := &corev1.Secret{}
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-db-configmap", sde.Namespace), Namespace: sde.Namespace}, configMap)
	if err != nil {
		return err
	}

	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-database-secrets", sde.Namespace), Namespace: sde.Namespace}, dbSecret)
	if err != nil {
		return err
	}

	conn := PGConnector{
		Host:     configMap.Data["DATABASE_HOST"],
		Port:     configMap.Data["DATABASE_PORT"],
		Password: string(dbSecret.Data["ADMIN_DATABASE_PASSWORD"]),
		User:     configMap.Data["ADMIN_DATABASE_USER"],
		Dbname:   "sde_",
		Sslmode:  false,
	}

	db, err := conn.Connect()
	if err != nil {
		return err
	}
	defer db.Close()

	// Query list of databases
	dbList := make([]string, 0)
	rows, err := db.Query(`SELECT datname FROM pg_database;`)
	if err != nil {
		return err
	}

	var dbName string
	for rows.Next() {
		if err := rows.Scan(dbName); err != nil {
			return err
		}
		dbList = append(dbList, dbName)
	}

	ctxlog.Info(fmt.Sprintf("yo sick %v", dbList))
	return nil
}
