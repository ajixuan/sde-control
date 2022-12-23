package controllers

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"regexp"

	"github.com/go-logr/logr"
	ver "github.com/hashicorp/go-version"
	_ "github.com/lib/pq"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	sdev1beta1 "sde.domain/sdeController/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var ctxlog logr.Logger
var re = regexp.MustCompile(`(\d+\.)?(\d+\.)?(\*|\d+)`)

type DbVersions []string

func (s DbVersions) Len() int {
	return len(s)
}

func (s DbVersions) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s DbVersions) Less(i, j int) bool {
	v1, err := ver.NewVersion(re.FindStringSubmatch(s[i])[0])
	if err != nil {
		v1, _ = ver.NewVersion("0")
	}

	v2, err := ver.NewVersion(re.FindStringSubmatch(s[j])[0])
	if err != nil {
		v2, _ = ver.NewVersion("0")
	}

	return v1.LessThan(v2)
}

type sslMode bool

func (s sslMode) String() string {
	switch s {
	case true:
		return "enabled"
	case false:
		return "disable"
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

func cleanupDB(db *sql.DB, dbList []string, count int) error {
	var query strings.Builder

	for i := 0; i < count; i++ {
		query.WriteString("DROP DATABASE ")
		query.WriteString("\"")
		query.WriteString(dbList[i])
		query.WriteString("\" ;")

		_, err := db.Exec(query.String())
		if err != nil {
			return err
		}

		query.Reset()
	}

	return nil
}

func (r *SdeReconciler) reconcileDb(ctx context.Context, sde *sdev1beta1.Sde) error {
	ctxlog = log.FromContext(ctx)
	ctxlog.Info("Reconciling Database...")

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
	rows, err := db.Query(`SELECT datname FROM pg_database WHERE datname ~ '^sde_[0-9]+\.[0-9]+\.[0-9]+.*$';`)
	if err != nil {
		return err
	}

	var dbName string
	for rows.Next() {
		if err := rows.Scan(&dbName); err != nil {
			return err
		}
		dbList = append(dbList, dbName)
	}

	sort.Sort(DbVersions(dbList))
	ctxlog.Info(fmt.Sprintf("Current DBs: %v", dbList))

	count := len(dbList) - int(sde.Spec.DatabaseCount)
	if count > 0 {
		err = cleanupDB(db, dbList, count)
		if err != nil {
			return err
		}
	}

	return nil
}
