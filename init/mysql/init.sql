-- 基于 ent schema 的 MySQL 初始化示例数据（对齐当前运行期迁移行为）
-- 执行方式：
--   mysql -uroot -p < init/mysql/init.sql

SET NAMES utf8mb4;
SET time_zone = '+00:00';

CREATE DATABASE IF NOT EXISTS `aihub`
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_bin;

USE `aihub`;

-- 0) 表结构初始化
-- 说明：
-- 1. 数值字段使用 BIGINT，避免 ent 迁移时将 INT -> BIGINT 引发冲突
-- 2. 不预建外键，和当前代码中的 migrate.WithForeignKeys(false) 保持一致

CREATE TABLE IF NOT EXISTS `roles` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` BIGINT NOT NULL DEFAULT 0,
  `code` VARCHAR(255) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `scopes` JSON NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `roles_by_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` BIGINT NOT NULL DEFAULT 0,
  `email` VARCHAR(255) NOT NULL,
  `status` ENUM('activated', 'deactivated') NOT NULL DEFAULT 'activated',
  `prefer_language` VARCHAR(255) NOT NULL DEFAULT 'en',
  `password` VARCHAR(255) NOT NULL,
  `first_name` VARCHAR(255) NOT NULL DEFAULT '',
  `last_name` VARCHAR(255) NOT NULL DEFAULT '',
  `avatar` VARCHAR(255) NULL,
  `is_owner` BOOLEAN NOT NULL DEFAULT FALSE,
  `scopes` JSON NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `users_email_key` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `channels` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` BIGINT NOT NULL DEFAULT 0,
  `type` ENUM(
    'openai', 'anthropic', 'anthropic_aws', 'anthropic_gcp',
    'deepseek', 'deepseek_anthropic', 'doubao', 'moonshot',
    'moonshot_anthropic', 'zhipu', 'zai', 'zhipu_anthropic',
    'zai_anthropic', 'anthropic_fake', 'openai_fake'
  ) NOT NULL,
  `base_url` VARCHAR(255) NULL,
  `name` VARCHAR(255) NOT NULL,
  `status` ENUM('enabled', 'disabled', 'archived') NOT NULL DEFAULT 'disabled',
  `credentials` JSON NOT NULL,
  `supported_models` JSON NOT NULL,
  `default_test_model` VARCHAR(255) NOT NULL,
  `settings` JSON NULL,
  `ordering_weight` BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `channels_by_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `api_keys` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` BIGINT NOT NULL DEFAULT 0,
  `key` VARCHAR(255) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `status` ENUM('enabled', 'disabled') NOT NULL DEFAULT 'enabled',
  `scopes` JSON NULL,
  `profiles` JSON NULL,
  `user_id` BIGINT NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `api_keys_by_key` (`key`),
  KEY `api_keys_by_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `requests` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` BIGINT NOT NULL DEFAULT 0,
  `source` ENUM('api', 'playground', 'test') NOT NULL DEFAULT 'api',
  `model_id` VARCHAR(255) NOT NULL,
  `format` VARCHAR(255) NOT NULL DEFAULT 'openai/chat_completions',
  `request_body` JSON NOT NULL,
  `response_body` JSON NULL,
  `response_chunks` JSON NULL,
  `external_id` VARCHAR(255) NULL,
  `status` ENUM('pending', 'processing', 'completed', 'failed') NOT NULL,
  `api_key_id` BIGINT NULL,
  `channel_id` BIGINT NULL,
  `user_id` BIGINT NOT NULL,
  PRIMARY KEY (`id`),
  KEY `requests_by_user_id` (`user_id`),
  KEY `requests_by_api_key_id` (`api_key_id`),
  KEY `requests_by_channel_id` (`channel_id`),
  KEY `requests_by_created_at` (`created_at`),
  KEY `requests_by_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `request_executions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user_id` BIGINT NOT NULL,
  `external_id` VARCHAR(255) NULL,
  `model_id` VARCHAR(255) NOT NULL,
  `format` VARCHAR(255) NOT NULL DEFAULT 'openai/chat_completions',
  `request_body` JSON NOT NULL,
  `response_body` JSON NULL,
  `response_chunks` JSON NULL,
  `error_message` VARCHAR(255) NULL,
  `status` ENUM('pending', 'processing', 'completed', 'failed') NOT NULL,
  `channel_id` BIGINT NOT NULL,
  `request_id` BIGINT NOT NULL,
  PRIMARY KEY (`id`),
  KEY `request_executions_by_request_id` (`request_id`),
  KEY `request_executions_by_channel_id_created_at` (`channel_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `usage_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `channel_usage_logs` BIGINT NULL,
  `request_usage_logs` BIGINT NULL,
  `user_usage_logs` BIGINT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `user_roles` (
  `user_id` BIGINT NOT NULL,
  `role_id` BIGINT NOT NULL,
  PRIMARY KEY (`user_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

START TRANSACTION;

-- 1) 角色
INSERT INTO `roles`
(`id`, `created_at`, `updated_at`, `deleted_at`, `code`, `name`, `scopes`)
VALUES
  (1, NOW(), NOW(), 0, 'admin', '管理员',
   '["read_channels","write_channels","read_users","add_users","write_requests"]'),
  (2, NOW(), NOW(), 0, 'developer', '开发者',
   '["read_channels","write_requests"]')
ON DUPLICATE KEY UPDATE
  `updated_at` = NOW(),
  `deleted_at` = 0,
  `name` = VALUES(`name`),
  `scopes` = VALUES(`scopes`);

-- 2) 用户
INSERT INTO `users`
(`id`, `created_at`, `updated_at`, `deleted_at`, `email`, `status`,
 `prefer_language`, `password`, `first_name`, `last_name`, `avatar`,
 `is_owner`, `scopes`)
VALUES
  (1, NOW(), NOW(), 0, 'owner@aihub.local', 'activated',
   'zh-CN', 'owner_demo_password_hash', '系统', '管理员', NULL, 1,
   '["read_channels","write_channels","read_users","add_users","write_requests"]'),
  (2, NOW(), NOW(), 0, 'dev@aihub.local', 'activated',
   'zh-CN', 'dev_demo_password_hash', '示例', '开发者', NULL, 0,
   '["read_channels","write_requests"]')
ON DUPLICATE KEY UPDATE
  `updated_at` = NOW(),
  `deleted_at` = 0,
  `status` = VALUES(`status`),
  `prefer_language` = VALUES(`prefer_language`),
  `password` = VALUES(`password`),
  `first_name` = VALUES(`first_name`),
  `last_name` = VALUES(`last_name`),
  `avatar` = VALUES(`avatar`),
  `is_owner` = VALUES(`is_owner`),
  `scopes` = VALUES(`scopes`);

-- 3) 用户-角色关联（多对多）
INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`) VALUES
  (1, 1),
  (2, 2);

-- 4) 渠道
INSERT INTO `channels`
(`id`, `created_at`, `updated_at`, `deleted_at`, `type`, `base_url`, `name`,
 `status`, `credentials`, `supported_models`, `default_test_model`,
 `settings`, `ordering_weight`)
VALUES
  (1, NOW(), NOW(), 0, 'openai', 'https://apis.iflow.cn/v1',
   'xinliu', 'enabled',
   '{"apiKey":"sk-a87ab03fc02b704c336b7ce1cb572588"}',
   '["qwen3-max","glm-4.6"]',
   'qwen3-max',
   '{"modelMappings":[{"from":"qwen3-max","to":"qwen3-max"},{"from":"glm-4.6","to":"glm-4.6"}]}',
   95),
  (2, NOW(), NOW(), 0, 'openai', 'http://127.0.0.1:11434/v1',
   'local-ollama', 'enabled',
   '{"apiKey":"local_123456"}',
   '["qwen3:0.6b"]',
   'qwen3:0.6b',
   '{"modelMappings":[{"from":"qwen3:0.6b","to":"qwen3:0.6b"}]}',
   88)
ON DUPLICATE KEY UPDATE
  `updated_at` = NOW(),
  `deleted_at` = 0,
  `base_url` = VALUES(`base_url`),
  `status` = VALUES(`status`),
  `credentials` = VALUES(`credentials`),
  `supported_models` = VALUES(`supported_models`),
  `default_test_model` = VALUES(`default_test_model`),
  `settings` = VALUES(`settings`),
  `ordering_weight` = VALUES(`ordering_weight`);

-- 5) API Key
INSERT INTO `api_keys`
(`id`, `created_at`, `updated_at`, `deleted_at`, `key`, `name`, `status`,
 `scopes`, `profiles`, `user_id`)
VALUES
  (1, NOW(), NOW(), 0, 'ak_demo_owner_001', 'Owner Default Key', 'enabled',
   '["read_channels","write_requests"]',
   '{"activeProfile":"default","profiles":[{"name":"default","modelMappings":[{"from":"gpt-4o-mini","to":"gpt-4o-mini"}]}]}',
   1),
  (2, NOW(), NOW(), 0, 'ak_demo_dev_001', 'Dev Default Key', 'enabled',
   '["read_channels","write_requests"]',
   '{"activeProfile":"default","profiles":[{"name":"default","modelMappings":[{"from":"gpt-4o-mini","to":"deepseek-chat"}]}]}',
   2)
ON DUPLICATE KEY UPDATE
  `updated_at` = NOW(),
  `deleted_at` = 0,
  `name` = VALUES(`name`),
  `status` = VALUES(`status`),
  `scopes` = VALUES(`scopes`),
  `profiles` = VALUES(`profiles`),
  `user_id` = VALUES(`user_id`);

-- 6) 请求（面向用户）
INSERT INTO `requests`
(`id`, `created_at`, `updated_at`, `deleted_at`, `source`, `model_id`, `format`,
 `request_body`, `response_body`, `response_chunks`, `external_id`, `status`,
 `api_key_id`, `channel_id`, `user_id`)
VALUES
  (1, NOW(), NOW(), 0, 'api', 'gpt-4o-mini', 'openai/chat_completions',
   '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"你好，给我一句问候语"}],"stream":false}',
   '{"id":"chatcmpl-demo-1","object":"chat.completion","model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"你好，欢迎使用 AIHub 示例数据。"},"finish_reason":"stop"}]}',
   '[]', 'req_demo_001', 'completed',
   1, 1, 1),
  (2, NOW(), NOW(), 0, 'test', 'deepseek-chat', 'openai/chat_completions',
   '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"请用一句话介绍你自己"}],"stream":true}',
   NULL,
   '[{"event":"message","data":"{\\"id\\":\\"chatcmpl-demo-2\\",\\"choices\\":[{\\"delta\\":{\\"content\\":\\"我是示例流式响应\\"}}]}"}]',
   'req_demo_002', 'completed',
   2, 2, 2)
ON DUPLICATE KEY UPDATE
  `updated_at` = NOW(),
  `deleted_at` = 0,
  `source` = VALUES(`source`),
  `model_id` = VALUES(`model_id`),
  `format` = VALUES(`format`),
  `request_body` = VALUES(`request_body`),
  `response_body` = VALUES(`response_body`),
  `response_chunks` = VALUES(`response_chunks`),
  `external_id` = VALUES(`external_id`),
  `status` = VALUES(`status`),
  `api_key_id` = VALUES(`api_key_id`),
  `channel_id` = VALUES(`channel_id`),
  `user_id` = VALUES(`user_id`);

-- 7) 请求执行（面向具体渠道）
INSERT INTO `request_executions`
(`id`, `created_at`, `updated_at`, `user_id`, `request_id`, `channel_id`,
 `external_id`, `model_id`, `format`, `request_body`, `response_body`,
 `response_chunks`, `error_message`, `status`)
VALUES
  (1, NOW(), NOW(), 1, 1, 1, 'exec_demo_001', 'gpt-4o-mini',
   'openai/chat_completions',
   '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"你好，给我一句问候语"}],"stream":false}',
   '{"id":"chatcmpl-demo-1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"你好，欢迎使用 AIHub 示例数据。"},"finish_reason":"stop"}]}',
   '[]', NULL, 'completed'),
  (2, NOW(), NOW(), 2, 2, 2, 'exec_demo_002', 'deepseek-chat',
   'openai/chat_completions',
   '{"model":"deepseek-chat","messages":[{"role":"user","content":"请用一句话介绍你自己"}],"stream":true}',
   NULL,
   '[{"event":"message","data":"{\\"choices\\":[{\\"delta\\":{\\"content\\":\\"我是示例流式响应\\"}}]}"}]',
   NULL, 'completed')
ON DUPLICATE KEY UPDATE
  `updated_at` = NOW(),
  `user_id` = VALUES(`user_id`),
  `request_id` = VALUES(`request_id`),
  `channel_id` = VALUES(`channel_id`),
  `external_id` = VALUES(`external_id`),
  `model_id` = VALUES(`model_id`),
  `format` = VALUES(`format`),
  `request_body` = VALUES(`request_body`),
  `response_body` = VALUES(`response_body`),
  `response_chunks` = VALUES(`response_chunks`),
  `error_message` = VALUES(`error_message`),
  `status` = VALUES(`status`);

-- 8) 用量日志（当前 schema 仅包含 3 个可空外键字段）
INSERT INTO `usage_logs`
(`id`, `channel_usage_logs`, `request_usage_logs`, `user_usage_logs`)
VALUES
  (1, 1, 1, 1),
  (2, 2, 2, 2)
ON DUPLICATE KEY UPDATE
  `channel_usage_logs` = VALUES(`channel_usage_logs`),
  `request_usage_logs` = VALUES(`request_usage_logs`),
  `user_usage_logs` = VALUES(`user_usage_logs`);

COMMIT;
