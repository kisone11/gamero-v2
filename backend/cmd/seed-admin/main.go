package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/pkg/database"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	email := flag.String("email", "admin@gamero.local", "管理员邮箱")
	password := flag.String("password", "Gamero123", "管理员密码")
	username := flag.String("username", "admin", "管理员用户名")
	nickname := flag.String("nickname", "Admin", "管理员昵称")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}
	if err := logger.Init(cfg.Logger); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if _, err := database.Init(cfg.Database); err != nil {
		logger.Fatal("数据库连接失败", zap.Error(err))
	}
	defer database.Close()

	if err := seedAdmin(database.Get(), *email, *password, *username, *nickname); err != nil {
		logger.Fatal("管理员账号插入失败", zap.Error(err))
	}
	logger.Info("管理员账号已就绪", zap.String("email", *email), zap.String("username", *username))
}

func seedAdmin(db *gorm.DB, email string, password string, username string, nickname string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := model.User{
		Username:     username,
		Nickname:     nickname,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         model.UserRoleSuperAdmin,
		Status:       model.UserStatusActive,
		IsBanned:     false,
	}

	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "email"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"username":      admin.Username,
			"nickname":      admin.Nickname,
			"password_hash": admin.PasswordHash,
			"role":          admin.Role,
			"status":        admin.Status,
			"is_banned":     false,
		}),
	}).Create(&admin).Error
}
