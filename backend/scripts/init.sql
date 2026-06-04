-- Gamero 数据库初始化脚本
-- 由 docker-compose 在 postgres 容器首次启动时自动执行
-- 此脚本仅做基础初始化，业务表结构由 GORM AutoMigrate 管理

-- 确保数据库编码正确（容器启动时已通过环境变量设置）
-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";   -- UUID 生成
CREATE EXTENSION IF NOT EXISTS "pg_trgm";     -- 中英文模糊搜索（trigram）
CREATE EXTENSION IF NOT EXISTS "unaccent";    -- 去除重音符号（搜索增强）

-- 设置时区
SET timezone = 'Asia/Shanghai';
